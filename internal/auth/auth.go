package auth

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// HMACSignature returns a hex-encoded HMAC-SHA256 of toSign using secret.
func HMACSignature(toSign, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(toSign))
	return hex.EncodeToString(mac.Sum(nil))
}

// CheckSignature verifies that signature is the HMAC-SHA256 of body using secret.
func CheckSignature(signature, secret string, body []byte) bool {
	expected := hmac.New(sha256.New, []byte(secret))
	expected.Write(body)
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(expected.Sum(nil), sigBytes)
}

// CreateAuthMap returns the auth map used in channel authorization responses.
// If sharedSecret is non-empty it is included under the "shared_secret" key.
func CreateAuthMap(key, secret, stringToSign, sharedSecret string) map[string]string {
	authString := strings.Join([]string{key, HMACSignature(stringToSign, secret)}, ":")
	if sharedSecret != "" {
		return map[string]string{"auth": authString, "shared_secret": sharedSecret}
	}
	return map[string]string{"auth": authString}
}

// MD5Hex returns the hex-encoded MD5 hash of body.
// MD5 is required by the Pusher signing protocol (body_md5 field), not a security choice.
func MD5Hex(body []byte) string {
	h := md5.New()
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
