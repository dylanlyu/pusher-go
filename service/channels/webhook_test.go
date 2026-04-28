package channels_test

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"golang.org/x/crypto/nacl/secretbox"

	"github.com/dylanlyu/pusher-go/service/channels"
)

// buildEncryptedEventData builds a base64+nacl-encrypted event payload
// using the same algorithm as channels/crypto.go.
func buildEncryptedEventData(channelName string, masterKey, plaintext []byte) (string, error) {
	combined := append([]byte(channelName), masterKey...)
	sharedSecret := sha256.Sum256(combined)

	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return "", err
	}

	ciphertext := secretbox.Seal(nil, plaintext, &nonce, &sharedSecret)
	msg := struct {
		Nonce      string `json:"nonce"`
		Ciphertext string `json:"ciphertext"`
	}{
		Nonce:      base64.StdEncoding.EncodeToString(nonce[:]),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	b, err := json.Marshal(msg)
	return string(b), err
}

func TestWebhook_DecryptEncryptedEvent(t *testing.T) {
	masterKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
		t.Fatal(err)
	}
	masterKeyB64 := base64.StdEncoding.EncodeToString(masterKey)

	channelName := "private-encrypted-test"
	plaintext := []byte(`{"msg":"secret hello"}`)

	encData, err := buildEncryptedEventData(channelName, masterKey, plaintext)
	if err != nil {
		t.Fatalf("failed to build encrypted event: %v", err)
	}

	// Build the webhook body with an encrypted event.
	webhookPayload := map[string]any{
		"time_ms": 1234567890,
		"events": []map[string]any{
			{
				"name":    "client-event",
				"channel": channelName,
				"event":   "my-event",
				"data":    encData,
			},
		},
	}
	body, err := json.Marshal(webhookPayload)
	if err != nil {
		t.Fatalf("marshal webhook payload: %v", err)
	}

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	c, _ := channels.New("123", "key", "secret",
		channels.WithEncryptionMasterKeyBase64(masterKeyB64),
	)

	header := http.Header{}
	header.Set("X-Pusher-Key", "key")
	header.Set("X-Pusher-Signature", sig)

	wh, err := c.Webhook(header, body)
	if err != nil {
		t.Fatalf("Webhook() unexpected error: %v", err)
	}
	if len(wh.Events) != 1 {
		t.Fatalf("Webhook() returned %d events, want 1", len(wh.Events))
	}
	if wh.Events[0].Data != string(plaintext) {
		t.Errorf("decrypted data = %q, want %q", wh.Events[0].Data, string(plaintext))
	}
}

func TestWebhook_EncryptedChannel_NoMasterKey(t *testing.T) {
	masterKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
		t.Fatalf("generate master key: %v", err)
	}

	channelName := "private-encrypted-test"
	encData, _ := buildEncryptedEventData(channelName, masterKey, []byte("hello"))

	webhookPayload := map[string]any{
		"time_ms": 1234567890,
		"events": []map[string]any{
			{"name": "client-event", "channel": channelName, "data": encData},
		},
	}
	body, _ := json.Marshal(webhookPayload)

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	// Client without encryption key.
	c, _ := channels.New("123", "key", "secret")

	header := http.Header{}
	header.Set("X-Pusher-Key", "key")
	header.Set("X-Pusher-Signature", sig)

	_, err := c.Webhook(header, body)
	if err == nil {
		t.Error("Webhook() expected error for encrypted channel with no master key")
	}
}
