package channels_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dylanlyu/pusher-go/service/channels"
)

// roundTripFunc is a transport-level mock that avoids real network calls.
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

// hmacSig computes HMAC-SHA256 for webhook test setup.
func hmacSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		appID   string
		key     string
		secret  string
		wantErr bool
	}{
		{"valid", "123", "key", "secret", false},
		{"missing appID", "", "key", "secret", true},
		{"missing key", "123", "", "secret", true},
		{"missing secret", "123", "key", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := channels.New(tc.appID, tc.key, tc.secret)
			if (err != nil) != tc.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestTrigger_Success(t *testing.T) {
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(200, "{}")),
	)
	err := c.Trigger(context.Background(), "my-channel", "test-event", map[string]string{"hello": "world"})
	if err != nil {
		t.Errorf("Trigger() unexpected error: %v", err)
	}
}

func TestTrigger_InvalidChannel(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	err := c.Trigger(context.Background(), "invalid channel!", "event", nil)
	if err == nil {
		t.Error("Trigger() expected error for invalid channel name")
	}
}

func TestTrigger_HTTPError(t *testing.T) {
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(400, `{"error":"bad request"}`)),
	)
	err := c.Trigger(context.Background(), "my-channel", "event", nil)
	if err == nil {
		t.Error("Trigger() expected error on HTTP 400")
	}
}

func TestTriggerMulti_TooManyChannels(t *testing.T) {
	chs := make([]string, 101)
	for i := range chs {
		chs[i] = fmt.Sprintf("ch-%d", i)
	}
	c, _ := channels.New("123", "key", "secret")
	err := c.TriggerMulti(context.Background(), chs, "event", nil)
	if err == nil {
		t.Error("TriggerMulti() expected error for >100 channels")
	}
}

func TestTriggerBatch_Success(t *testing.T) {
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(200, `{"batch":[]}`)),
	)
	batch := []channels.Event{
		{Channel: "ch-1", Name: "ev1", Data: "d1"},
		{Channel: "ch-2", Name: "ev2", Data: "d2"},
	}
	_, err := c.TriggerBatch(context.Background(), batch)
	if err != nil {
		t.Errorf("TriggerBatch() unexpected error: %v", err)
	}
}

func TestTriggerBatch_InvalidChannel(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	batch := []channels.Event{
		{Channel: "bad channel!", Name: "ev", Data: "d"},
	}
	_, err := c.TriggerBatch(context.Background(), batch)
	if err == nil {
		t.Error("TriggerBatch() expected error for invalid channel name")
	}
}

func TestAuthorizePrivateChannel(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	params := []byte("socket_id=1234.56&channel_name=private-chat")
	resp, err := c.AuthorizePrivateChannel(params)
	if err != nil {
		t.Fatalf("AuthorizePrivateChannel() unexpected error: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(resp, &m); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := m["auth"]; !ok {
		t.Error("response missing 'auth' key")
	}
}

func TestAuthorizePrivateChannel_MissingSocketID(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	params := []byte("channel_name=private-chat")
	_, err := c.AuthorizePrivateChannel(params)
	if err == nil {
		t.Error("AuthorizePrivateChannel() expected error for missing socket_id")
	}
}

func TestAuthorizePresenceChannel(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	params := []byte("socket_id=1234.56&channel_name=presence-room")
	member := channels.MemberData{UserID: "user1", UserInfo: map[string]string{"name": "Alice"}}
	resp, err := c.AuthorizePresenceChannel(params, member)
	if err != nil {
		t.Fatalf("AuthorizePresenceChannel() unexpected error: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(resp, &m); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := m["auth"]; !ok {
		t.Error("response missing 'auth' key")
	}
	if _, ok := m["channel_data"]; !ok {
		t.Error("response missing 'channel_data' key")
	}
}

func TestWebhook_ValidSignature(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	body := []byte(`{"time_ms":1234567890,"events":[{"name":"channel_occupied","channel":"test"}]}`)
	sig := hmacSig("secret", body)

	header := http.Header{}
	header.Set("X-Pusher-Key", "key")
	header.Set("X-Pusher-Signature", sig)

	wh, err := c.Webhook(header, body)
	if err != nil {
		t.Fatalf("Webhook() unexpected error: %v", err)
	}
	if wh == nil || len(wh.Events) != 1 {
		t.Errorf("Webhook() returned unexpected result: %+v", wh)
	}
}

func TestWebhook_InvalidSignature(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	body := []byte(`{"time_ms":1234567890,"events":[]}`)
	header := http.Header{}
	header.Set("X-Pusher-Key", "key")
	header.Set("X-Pusher-Signature", "invalidsignature")
	_, err := c.Webhook(header, body)
	if err == nil {
		t.Error("Webhook() expected error for invalid signature")
	}
}
