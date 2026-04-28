package channels_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dylanlyu/pusher-go/channels"
)

func TestTriggerWithParams_Success(t *testing.T) {
	socketID := "1234.56"
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(200, `{"channels":{}}`)),
	)
	_, err := c.TriggerWithParams(context.Background(), "my-channel", "event",
		map[string]string{"data": "value"},
		channels.TriggerParams{SocketID: &socketID},
	)
	if err != nil {
		t.Errorf("TriggerWithParams() unexpected error: %v", err)
	}
}

func TestTriggerMultiWithParams_Success(t *testing.T) {
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(200, `{"channels":{}}`)),
	)
	_, err := c.TriggerMultiWithParams(context.Background(),
		[]string{"ch-1", "ch-2"}, "event", "data", channels.TriggerParams{},
	)
	if err != nil {
		t.Errorf("TriggerMultiWithParams() unexpected error: %v", err)
	}
}

func TestSendToUser_Success(t *testing.T) {
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(200, `{}`)),
	)
	err := c.SendToUser(context.Background(), "user123", "event", map[string]string{"msg": "hi"})
	if err != nil {
		t.Errorf("SendToUser() unexpected error: %v", err)
	}
}

func TestSendToUser_InvalidUserID(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	err := c.SendToUser(context.Background(), "", "event", nil)
	if err == nil {
		t.Error("SendToUser() expected error for empty user ID")
	}
}

func TestChannelsList(t *testing.T) {
	body := `{"channels":{"presence-room":{"user_count":3}}}`
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(200, body)),
	)
	list, err := c.Channels(context.Background(), channels.ChannelsParams{})
	if err != nil {
		t.Fatalf("Channels() unexpected error: %v", err)
	}
	if len(list.Channels) != 1 {
		t.Errorf("Channels() returned %d channels, want 1", len(list.Channels))
	}
}

func TestChannelInfo(t *testing.T) {
	body := `{"occupied":true,"user_count":5}`
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(200, body)),
	)
	ch, err := c.Channel(context.Background(), "presence-room", channels.ChannelParams{})
	if err != nil {
		t.Fatalf("Channel() unexpected error: %v", err)
	}
	if !ch.Occupied {
		t.Error("Channel() returned channel not occupied")
	}
}

func TestGetChannelUsers(t *testing.T) {
	body := `{"users":[{"id":"u1"},{"id":"u2"}]}`
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(200, body)),
	)
	users, err := c.GetChannelUsers(context.Background(), "presence-room")
	if err != nil {
		t.Fatalf("GetChannelUsers() unexpected error: %v", err)
	}
	if len(users.List) != 2 {
		t.Errorf("GetChannelUsers() returned %d users, want 2", len(users.List))
	}
}

func TestAuthenticateUser(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	params := []byte("socket_id=1234.56")
	userData := map[string]any{"id": "user1", "name": "Alice"}
	resp, err := c.AuthenticateUser(params, userData)
	if err != nil {
		t.Fatalf("AuthenticateUser() unexpected error: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(resp, &m); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := m["auth"]; !ok {
		t.Error("response missing 'auth' key")
	}
	if _, ok := m["user_data"]; !ok {
		t.Error("response missing 'user_data' key")
	}
}

func TestAuthenticateUser_MissingID(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	params := []byte("socket_id=1234.56")
	_, err := c.AuthenticateUser(params, map[string]any{"name": "Alice"})
	if err == nil {
		t.Error("AuthenticateUser() expected error when user data missing id")
	}
}

func TestOptionsSetCluster(t *testing.T) {
	c, err := channels.New("123", "key", "secret",
		channels.WithCluster("eu"),
		channels.WithSecure(true),
		channels.WithHTTPClient(mockHTTPClient(200, `{}`)),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	// Just verify the client is created without error.
	_ = c
}

func TestOptionsSetHost(t *testing.T) {
	c, err := channels.New("123", "key", "secret",
		channels.WithHost("custom.host.com"),
		channels.WithHTTPClient(mockHTTPClient(200, `{}`)),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	_ = c
}

func TestOptionsSetMaxPayload(t *testing.T) {
	c, err := channels.New("123", "key", "secret",
		channels.WithMaxMessagePayloadKB(5),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	_ = c
}

func TestWithEncryptionMasterKey_Invalid(t *testing.T) {
	_, err := channels.New("123", "key", "secret",
		channels.WithEncryptionMasterKeyBase64("not-valid-base64!!!"),
	)
	if err == nil {
		t.Error("New() expected error for invalid base64 encryption key")
	}
}

func TestWithEncryptionMasterKey_WrongLength(t *testing.T) {
	_, err := channels.New("123", "key", "secret",
		channels.WithEncryptionMasterKeyBase64("dGVzdA=="), // "test" = 4 bytes
	)
	if err == nil {
		t.Error("New() expected error for encryption key that is not 32 bytes")
	}
}

func TestTrigger_InvalidSocketID(t *testing.T) {
	c, _ := channels.New("123", "key", "secret")
	badSocket := "not-a-socket-id"
	_, err := c.TriggerWithParams(context.Background(), "my-channel", "event", nil,
		channels.TriggerParams{SocketID: &badSocket},
	)
	if err == nil {
		t.Error("TriggerWithParams() expected error for invalid socket_id")
	}
}

func TestChannelsList_HTTPError(t *testing.T) {
	c, _ := channels.New("123", "key", "secret",
		channels.WithHTTPClient(mockHTTPClient(500, `{"error":"server error"}`)),
	)
	_, err := c.Channels(context.Background(), channels.ChannelsParams{})
	if err == nil {
		t.Error("Channels() expected error on HTTP 500")
	}
}
