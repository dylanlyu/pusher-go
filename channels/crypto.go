package channels

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

// generateSharedSecret derives a per-channel 32-byte shared secret.
func generateSharedSecret(channelName string, masterKey []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte(channelName))
	h.Write(masterKey)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func generateNonce() ([24]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nonce, err
	}
	return nonce, nil
}

func encryptData(channelName string, data []byte, masterKey []byte) (string, error) {
	sharedSecret := generateSharedSecret(channelName, masterKey)
	nonce, err := generateNonce()
	if err != nil {
		return "", err
	}
	ciphertext := secretbox.Seal(nil, data, &nonce, &sharedSecret)
	msg := EncryptedMessage{
		Nonce:      base64.StdEncoding.EncodeToString(nonce[:]),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	out, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func decryptEvents(wh Webhook, masterKey []byte) (*Webhook, error) {
	result := &Webhook{TimeMs: wh.TimeMs}
	for _, ev := range wh.Events {
		if !isEncryptedChannel(ev.Channel) {
			result.Events = append(result.Events, ev)
			continue
		}
		var msg EncryptedMessage
		if err := json.Unmarshal([]byte(ev.Data), &msg); err != nil {
			return nil, err
		}
		cipherBytes, err := base64.StdEncoding.DecodeString(msg.Ciphertext)
		if err != nil {
			return nil, err
		}
		nonceBytes, err := base64.StdEncoding.DecodeString(msg.Nonce)
		if err != nil {
			return nil, err
		}
		var nonce [24]byte
		copy(nonce[:], nonceBytes)
		sharedSecret := generateSharedSecret(ev.Channel, masterKey)
		plain, ok := secretbox.Open(nil, cipherBytes, &nonce, &sharedSecret)
		if !ok {
			return nil, errors.New("channels: failed to decrypt event — wrong key?")
		}
		ev.Data = string(plain)
		result.Events = append(result.Events, ev)
	}
	return result, nil
}
