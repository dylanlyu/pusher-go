package beams_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dylanlyu/pusher-go/beams"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mockHTTPClient(statusCode int, body string) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
}

// --- New ---

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
		secretKey  string
		wantErr    bool
	}{
		{"valid", "instance-1", "secret-key", false},
		{"missing instanceID", "", "secret-key", true},
		{"missing secretKey", "instance-1", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := beams.New(tc.instanceID, tc.secretKey)
			if (err != nil) != tc.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// --- PublishToInterests ---

func TestPublishToInterests_Success(t *testing.T) {
	c, _ := beams.New("inst", "key",
		beams.WithHTTPClient(mockHTTPClient(200, `{"publishId":"pub-123"}`)),
	)
	id, err := c.PublishToInterests(context.Background(),
		[]string{"sports", "news"},
		map[string]any{"apns": map[string]any{"aps": map[string]any{"alert": "Hello"}}},
	)
	if err != nil {
		t.Fatalf("PublishToInterests() unexpected error: %v", err)
	}
	if id != "pub-123" {
		t.Errorf("publishID = %q, want %q", id, "pub-123")
	}
}

func TestPublishToInterests_EmptyInterests(t *testing.T) {
	c, _ := beams.New("inst", "key")
	_, err := c.PublishToInterests(context.Background(), []string{}, map[string]any{})
	if err == nil {
		t.Error("PublishToInterests() expected error for empty interests")
	}
}

func TestPublishToInterests_TooManyInterests(t *testing.T) {
	interests := make([]string, 101)
	for i := range interests {
		interests[i] = "interest"
	}
	c, _ := beams.New("inst", "key")
	_, err := c.PublishToInterests(context.Background(), interests, map[string]any{})
	if err == nil {
		t.Error("PublishToInterests() expected error for >100 interests")
	}
}

func TestPublishToInterests_InvalidInterestName(t *testing.T) {
	tests := []struct {
		name     string
		interest string
	}{
		{"empty interest", ""},
		{"too long", strings.Repeat("a", 165)},
		{"invalid char", "my interest!"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := beams.New("inst", "key")
			_, err := c.PublishToInterests(context.Background(), []string{tc.interest}, map[string]any{})
			if err == nil {
				t.Errorf("PublishToInterests() expected error for interest %q", tc.interest)
			}
		})
	}
}

func TestPublishToInterests_HTTPError(t *testing.T) {
	c, _ := beams.New("inst", "key",
		beams.WithHTTPClient(mockHTTPClient(400, `{"error":"Bad request","description":"invalid payload"}`)),
	)
	_, err := c.PublishToInterests(context.Background(), []string{"sports"}, map[string]any{})
	if err == nil {
		t.Error("PublishToInterests() expected error on HTTP 400")
	}
}

func TestPublishToInterests_DoesNotMutateBody(t *testing.T) {
	c, _ := beams.New("inst", "key",
		beams.WithHTTPClient(mockHTTPClient(200, `{"publishId":"x"}`)),
	)
	original := map[string]any{"apns": "data"}
	_, err := c.PublishToInterests(context.Background(), []string{"topic"}, original)
	if err != nil {
		t.Fatalf("PublishToInterests() unexpected error: %v", err)
	}
	if _, mutated := original["interests"]; mutated {
		t.Error("PublishToInterests() mutated the caller's body map")
	}
}

// --- PublishToUsers ---

func TestPublishToUsers_Success(t *testing.T) {
	c, _ := beams.New("inst", "key",
		beams.WithHTTPClient(mockHTTPClient(200, `{"publishId":"pub-456"}`)),
	)
	id, err := c.PublishToUsers(context.Background(),
		[]string{"user-1", "user-2"},
		map[string]any{"fcm": map[string]any{"notification": map[string]any{"title": "Hi"}}},
	)
	if err != nil {
		t.Fatalf("PublishToUsers() unexpected error: %v", err)
	}
	if id != "pub-456" {
		t.Errorf("publishID = %q, want %q", id, "pub-456")
	}
}

func TestPublishToUsers_EmptyUsers(t *testing.T) {
	c, _ := beams.New("inst", "key")
	_, err := c.PublishToUsers(context.Background(), []string{}, map[string]any{})
	if err == nil {
		t.Error("PublishToUsers() expected error for empty users")
	}
}

func TestPublishToUsers_TooManyUsers(t *testing.T) {
	users := make([]string, 1001)
	for i := range users {
		users[i] = "user"
	}
	c, _ := beams.New("inst", "key")
	_, err := c.PublishToUsers(context.Background(), users, map[string]any{})
	if err == nil {
		t.Error("PublishToUsers() expected error for >1000 users")
	}
}

func TestPublishToUsers_DoesNotMutateBody(t *testing.T) {
	c, _ := beams.New("inst", "key",
		beams.WithHTTPClient(mockHTTPClient(200, `{"publishId":"x"}`)),
	)
	original := map[string]any{"fcm": "data"}
	_, _ = c.PublishToUsers(context.Background(), []string{"u1"}, original)
	if _, mutated := original["users"]; mutated {
		t.Error("PublishToUsers() mutated the caller's body map")
	}
}

// --- GenerateToken ---

func TestGenerateToken_Success(t *testing.T) {
	c, _ := beams.New("inst", "key")
	result, err := c.GenerateToken("user123")
	if err != nil {
		t.Fatalf("GenerateToken() unexpected error: %v", err)
	}
	if _, ok := result["token"]; !ok {
		t.Error("GenerateToken() result missing 'token' key")
	}
}

func TestGenerateToken_EmptyUserID(t *testing.T) {
	c, _ := beams.New("inst", "key")
	_, err := c.GenerateToken("")
	if err == nil {
		t.Error("GenerateToken() expected error for empty userID")
	}
}

func TestGenerateToken_TooLongUserID(t *testing.T) {
	c, _ := beams.New("inst", "key")
	_, err := c.GenerateToken(strings.Repeat("a", 165))
	if err == nil {
		t.Error("GenerateToken() expected error for userID >164 chars")
	}
}

// --- DeleteUser ---

func TestDeleteUser_Success(t *testing.T) {
	c, _ := beams.New("inst", "key",
		beams.WithHTTPClient(mockHTTPClient(200, "")),
	)
	err := c.DeleteUser(context.Background(), "user123")
	if err != nil {
		t.Errorf("DeleteUser() unexpected error: %v", err)
	}
}

func TestDeleteUser_EmptyUserID(t *testing.T) {
	c, _ := beams.New("inst", "key")
	err := c.DeleteUser(context.Background(), "")
	if err == nil {
		t.Error("DeleteUser() expected error for empty userID")
	}
}

func TestDeleteUser_HTTPError(t *testing.T) {
	c, _ := beams.New("inst", "key",
		beams.WithHTTPClient(mockHTTPClient(404, `{"error":"Not found","description":"user not found"}`)),
	)
	err := c.DeleteUser(context.Background(), "ghost")
	if err == nil {
		t.Error("DeleteUser() expected error on HTTP 404")
	}
}

// --- Options ---

func TestWithBaseURL(t *testing.T) {
	c, err := beams.New("inst", "key",
		beams.WithBaseURL("http://localhost:8080"),
		beams.WithHTTPClient(mockHTTPClient(200, `{"publishId":"x"}`)),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	_, err = c.PublishToInterests(context.Background(), []string{"t"}, map[string]any{})
	if err != nil {
		t.Errorf("PublishToInterests() with custom base URL unexpected error: %v", err)
	}
}
