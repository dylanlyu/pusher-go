package channels

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

// Option configures a channels Client.
type Option func(*channelConfig)

// WithHTTPClient sets a custom HTTP client for all API requests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *channelConfig) { c.httpClient = hc }
}

// WithHost overrides the default API host (api.pusherapp.com).
func WithHost(host string) Option {
	return func(c *channelConfig) { c.host = host }
}

// WithCluster sets the Pusher cluster (e.g. "eu", "ap1").
// It is ignored if WithHost is also provided.
func WithCluster(cluster string) Option {
	return func(c *channelConfig) { c.cluster = cluster }
}

// WithSecure forces HTTPS for all API requests.
func WithSecure(secure bool) Option {
	return func(c *channelConfig) { c.secure = secure }
}

// WithEncryptionMasterKeyBase64 sets the 32-byte E2E encryption master key
// as a base64-encoded string. Required for encrypted channels.
func WithEncryptionMasterKeyBase64(key string) Option {
	return func(c *channelConfig) { c.encryptionMasterKeyB64 = key }
}

// WithMaxMessagePayloadKB overrides the default 10 KB payload limit.
func WithMaxMessagePayloadKB(kb int) Option {
	return func(c *channelConfig) { c.maxMessagePayloadKB = kb }
}

type channelConfig struct {
	appID                  string
	key                    string
	secret                 string
	host                   string
	cluster                string
	secure                 bool
	encryptionMasterKeyB64 string
	encryptionMasterKey    []byte // decoded and validated at New() time
	maxMessagePayloadKB    int
	httpClient             *http.Client
}

func (c *channelConfig) resolveEncryptionKey() error {
	if c.encryptionMasterKeyB64 == "" {
		return nil
	}
	keyBytes, err := base64.StdEncoding.DecodeString(c.encryptionMasterKeyB64)
	if err != nil {
		return fmt.Errorf("channels: EncryptionMasterKeyBase64 is not valid base64: %w", err)
	}
	if len(keyBytes) != 32 {
		return fmt.Errorf("channels: EncryptionMasterKeyBase64 must encode 32 bytes, got %d", len(keyBytes))
	}
	c.encryptionMasterKey = keyBytes
	return nil
}

func (c *channelConfig) httpClientOrDefault() *http.Client {
	if c.httpClient != nil {
		return c.httpClient
	}
	return http.DefaultClient
}
