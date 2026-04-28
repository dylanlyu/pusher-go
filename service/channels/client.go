package channels

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	internalauth "github.com/dylanlyu/pusher-go/internal/auth"
	"github.com/dylanlyu/pusher-go/internal/request"
)

const (
	libraryName    = "pusher-go"
	libraryVersion = "1.0.0"
	maxChannels    = 100
)

func defaultHeaders() map[string]string {
	return map[string]string{
		"Content-Type":     "application/json",
		"X-Pusher-Library": libraryName + " " + libraryVersion,
	}
}

// Client is the Pusher Channels HTTP API client interface.
type Client interface {
	Trigger(ctx context.Context, channel, event string, data any) error
	TriggerWithParams(ctx context.Context, channel, event string, data any, params TriggerParams) (*TriggerChannelsList, error)
	TriggerMulti(ctx context.Context, channels []string, event string, data any) error
	TriggerMultiWithParams(ctx context.Context, channels []string, event string, data any, params TriggerParams) (*TriggerChannelsList, error)
	TriggerBatch(ctx context.Context, batch []Event) (*TriggerBatchChannelsList, error)
	SendToUser(ctx context.Context, userID, event string, data any) error
	Channels(ctx context.Context, params ChannelsParams) (*ChannelsList, error)
	Channel(ctx context.Context, name string, params ChannelParams) (*Channel, error)
	GetChannelUsers(ctx context.Context, name string) (*Users, error)
	AuthorizePrivateChannel(params []byte) ([]byte, error)
	AuthorizePresenceChannel(params []byte, member MemberData) ([]byte, error)
	AuthenticateUser(params []byte, userData map[string]any) ([]byte, error)
	TerminateUserConnections(ctx context.Context, userID string) error
	Webhook(header http.Header, body []byte) (*Webhook, error)
}

type client struct {
	cfg channelConfig
}

// New constructs a Client. appID, key, and secret are required.
func New(appID, key, secret string, opts ...Option) (Client, error) {
	if appID == "" {
		return nil, errors.New("channels: appID is required")
	}
	if key == "" {
		return nil, errors.New("channels: key is required")
	}
	if secret == "" {
		return nil, errors.New("channels: secret is required")
	}

	cfg := channelConfig{appID: appID, key: key, secret: secret}
	for _, opt := range opts {
		opt(&cfg)
	}
	if err := cfg.resolveEncryptionKey(); err != nil {
		return nil, err
	}
	return &client{cfg: cfg}, nil
}

// --- Trigger methods ---

func (c *client) Trigger(ctx context.Context, channel, event string, data any) error {
	_, err := c.triggerChannels(ctx, []string{channel}, event, data, TriggerParams{})
	return err
}

func (c *client) TriggerWithParams(ctx context.Context, channel, event string, data any, params TriggerParams) (*TriggerChannelsList, error) {
	return c.triggerChannels(ctx, []string{channel}, event, data, params)
}

func (c *client) TriggerMulti(ctx context.Context, chs []string, event string, data any) error {
	_, err := c.triggerChannels(ctx, chs, event, data, TriggerParams{})
	return err
}

func (c *client) TriggerMultiWithParams(ctx context.Context, chs []string, event string, data any, params TriggerParams) (*TriggerChannelsList, error) {
	return c.triggerChannels(ctx, chs, event, data, params)
}

func (c *client) triggerChannels(ctx context.Context, chs []string, event string, data any, params TriggerParams) (*TriggerChannelsList, error) {
	if len(chs) > maxChannels {
		return nil, fmt.Errorf("channels: cannot trigger on more than %d channels at once", maxChannels)
	}
	if !channelsAreValid(chs) {
		return nil, errors.New("channels: one or more channel names are invalid")
	}

	hasEncrypted := false
	for _, ch := range chs {
		if isEncryptedChannel(ch) {
			hasEncrypted = true
			break
		}
	}
	if hasEncrypted && len(chs) > 1 {
		return nil, errors.New("channels: cannot trigger to multiple channels when using encrypted channels")
	}
	if hasEncrypted && c.cfg.encryptionMasterKey == nil {
		return nil, errors.New("channels: encryption master key required for encrypted channels")
	}

	if err := validateSocketID(params.SocketID); err != nil {
		return nil, err
	}

	payload, err := encodeTriggerBody(chs, event, data, params.toMap(), c.cfg.encryptionMasterKey, c.cfg.maxMessagePayloadKB)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/apps/%s/events", c.cfg.appID)
	u, err := buildRequestURL("POST", c.cfg.host, path, c.cfg.key, c.cfg.secret, c.cfg.secure, payload, nil, c.cfg.cluster)
	if err != nil {
		return nil, err
	}

	resp, err := request.Do(ctx, c.cfg.httpClientOrDefault(), "POST", u, payload, defaultHeaders())
	if err != nil {
		return nil, err
	}

	var list TriggerChannelsList
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, fmt.Errorf("channels: parse trigger response: %w", err)
	}
	return &list, nil
}

func (c *client) TriggerBatch(ctx context.Context, batch []Event) (*TriggerBatchChannelsList, error) {
	hasEncrypted := false
	for _, e := range batch {
		if !ValidChannel(e.Channel) {
			return nil, fmt.Errorf("channels: invalid channel name %q in batch", e.Channel)
		}
		if err := validateSocketID(e.SocketID); err != nil {
			return nil, err
		}
		if isEncryptedChannel(e.Channel) {
			hasEncrypted = true
			break
		}
	}
	if hasEncrypted && c.cfg.encryptionMasterKey == nil {
		return nil, errors.New("channels: encryption master key required for encrypted channels")
	}

	payload, err := encodeTriggerBatchBody(batch, c.cfg.encryptionMasterKey, c.cfg.maxMessagePayloadKB)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/apps/%s/batch_events", c.cfg.appID)
	u, err := buildRequestURL("POST", c.cfg.host, path, c.cfg.key, c.cfg.secret, c.cfg.secure, payload, nil, c.cfg.cluster)
	if err != nil {
		return nil, err
	}

	resp, err := request.Do(ctx, c.cfg.httpClientOrDefault(), "POST", u, payload, defaultHeaders())
	if err != nil {
		return nil, err
	}

	var list TriggerBatchChannelsList
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, fmt.Errorf("channels: parse batch trigger response: %w", err)
	}
	return &list, nil
}

func (c *client) SendToUser(ctx context.Context, userID, event string, data any) error {
	if !validUserID(userID) {
		return fmt.Errorf("channels: invalid user ID %q", userID)
	}
	// Bypass channel name validation — server-to-user channels use '#' prefix.
	_, err := c.triggerRaw(ctx, []string{"#server-to-user-" + userID}, event, data, TriggerParams{})
	return err
}

// triggerRaw sends events without validating channel names.
// Used for internal channels like #server-to-user-*.
func (c *client) triggerRaw(ctx context.Context, chs []string, event string, data any, params TriggerParams) (*TriggerChannelsList, error) {
	if err := validateSocketID(params.SocketID); err != nil {
		return nil, err
	}
	payload, err := encodeTriggerBody(chs, event, data, params.toMap(), nil, c.cfg.maxMessagePayloadKB)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/apps/%s/events", c.cfg.appID)
	u, err := buildRequestURL("POST", c.cfg.host, path, c.cfg.key, c.cfg.secret, c.cfg.secure, payload, nil, c.cfg.cluster)
	if err != nil {
		return nil, err
	}
	resp, err := request.Do(ctx, c.cfg.httpClientOrDefault(), "POST", u, payload, defaultHeaders())
	if err != nil {
		return nil, err
	}
	var list TriggerChannelsList
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, fmt.Errorf("channels: parse trigger response: %w", err)
	}
	return &list, nil
}

// --- Channel query methods ---

func (c *client) Channels(ctx context.Context, params ChannelsParams) (*ChannelsList, error) {
	path := fmt.Sprintf("/apps/%s/channels", c.cfg.appID)
	u, err := buildRequestURL("GET", c.cfg.host, path, c.cfg.key, c.cfg.secret, c.cfg.secure, nil, params.toMap(), c.cfg.cluster)
	if err != nil {
		return nil, err
	}
	resp, err := request.Do(ctx, c.cfg.httpClientOrDefault(), "GET", u, nil, defaultHeaders())
	if err != nil {
		return nil, err
	}
	var list ChannelsList
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, fmt.Errorf("channels: parse channels response: %w", err)
	}
	return &list, nil
}

func (c *client) Channel(ctx context.Context, name string, params ChannelParams) (*Channel, error) {
	path := fmt.Sprintf("/apps/%s/channels/%s", c.cfg.appID, name)
	u, err := buildRequestURL("GET", c.cfg.host, path, c.cfg.key, c.cfg.secret, c.cfg.secure, nil, params.toMap(), c.cfg.cluster)
	if err != nil {
		return nil, err
	}
	resp, err := request.Do(ctx, c.cfg.httpClientOrDefault(), "GET", u, nil, defaultHeaders())
	if err != nil {
		return nil, err
	}
	ch := &Channel{Name: name}
	if err := json.Unmarshal(resp, ch); err != nil {
		return nil, fmt.Errorf("channels: parse channel response: %w", err)
	}
	return ch, nil
}

func (c *client) GetChannelUsers(ctx context.Context, name string) (*Users, error) {
	path := fmt.Sprintf("/apps/%s/channels/%s/users", c.cfg.appID, name)
	u, err := buildRequestURL("GET", c.cfg.host, path, c.cfg.key, c.cfg.secret, c.cfg.secure, nil, nil, c.cfg.cluster)
	if err != nil {
		return nil, err
	}
	resp, err := request.Do(ctx, c.cfg.httpClientOrDefault(), "GET", u, nil, defaultHeaders())
	if err != nil {
		return nil, err
	}
	var users Users
	if err := json.Unmarshal(resp, &users); err != nil {
		return nil, fmt.Errorf("channels: parse users response: %w", err)
	}
	return &users, nil
}

func (c *client) TerminateUserConnections(ctx context.Context, userID string) error {
	if !validUserID(userID) {
		return fmt.Errorf("channels: invalid user ID %q", userID)
	}
	path := fmt.Sprintf("/apps/%s/users/%s/terminate_connections", c.cfg.appID, userID)
	u, err := buildRequestURL("POST", c.cfg.host, path, c.cfg.key, c.cfg.secret, c.cfg.secure, nil, nil, c.cfg.cluster)
	if err != nil {
		return err
	}
	_, err = request.Do(ctx, c.cfg.httpClientOrDefault(), "POST", u, nil, defaultHeaders())
	return err
}

// --- Authorization / Authentication ---

func (c *client) AuthorizePrivateChannel(params []byte) ([]byte, error) {
	return c.authorizeChannel(params, nil)
}

func (c *client) AuthorizePresenceChannel(params []byte, member MemberData) ([]byte, error) {
	return c.authorizeChannel(params, &member)
}

func (c *client) authorizeChannel(params []byte, member *MemberData) ([]byte, error) {
	channelName, socketID, err := parseChannelAuthParams(params)
	if err != nil {
		return nil, err
	}
	if err := validateSocketID(&socketID); err != nil {
		return nil, err
	}

	stringToSign := socketID + ":" + channelName
	var jsonUserData string

	if member != nil {
		memberBytes, err := json.Marshal(member)
		if err != nil {
			return nil, fmt.Errorf("channels: marshal member data: %w", err)
		}
		jsonUserData = string(memberBytes)
		stringToSign = stringToSign + ":" + jsonUserData
	}

	var authMap map[string]string
	if isEncryptedChannel(channelName) {
		if c.cfg.encryptionMasterKey == nil {
			return nil, errors.New("channels: encryption master key required for encrypted channels")
		}
		sharedSecret := generateSharedSecret(channelName, c.cfg.encryptionMasterKey)
		sharedSecretB64 := base64.StdEncoding.EncodeToString(sharedSecret[:])
		authMap = internalauth.CreateAuthMap(c.cfg.key, c.cfg.secret, stringToSign, sharedSecretB64)
	} else {
		authMap = internalauth.CreateAuthMap(c.cfg.key, c.cfg.secret, stringToSign, "")
	}

	if member != nil {
		authMap["channel_data"] = jsonUserData
	}
	return json.Marshal(authMap)
}

func (c *client) AuthenticateUser(params []byte, userData map[string]any) ([]byte, error) {
	socketID, err := parseUserAuthParams(params)
	if err != nil {
		return nil, err
	}
	if err := validateSocketID(&socketID); err != nil {
		return nil, err
	}
	if err := validateUserData(userData); err != nil {
		return nil, err
	}

	jsonUserData, err := json.Marshal(userData)
	if err != nil {
		return nil, fmt.Errorf("channels: marshal user data: %w", err)
	}

	stringToSign := strings.Join([]string{socketID, "user", string(jsonUserData)}, "::")
	authMap := internalauth.CreateAuthMap(c.cfg.key, c.cfg.secret, stringToSign, "")
	authMap["user_data"] = string(jsonUserData)
	return json.Marshal(authMap)
}

// --- Webhook ---

func (c *client) Webhook(header http.Header, body []byte) (*Webhook, error) {
	for _, token := range header["X-Pusher-Key"] {
		if token != c.cfg.key {
			continue
		}
		if !internalauth.CheckSignature(header.Get("X-Pusher-Signature"), c.cfg.secret, body) {
			return nil, errors.New("channels: webhook signature is invalid")
		}
		wh, err := parseWebhook(body)
		if err != nil {
			return nil, err
		}
		hasEncrypted := false
		for _, ev := range wh.Events {
			if isEncryptedChannel(ev.Channel) {
				hasEncrypted = true
				break
			}
		}
		if hasEncrypted {
			if c.cfg.encryptionMasterKey == nil {
				return nil, errors.New("channels: encryption master key required for encrypted channels")
			}
			return decryptEvents(*wh, c.cfg.encryptionMasterKey)
		}
		return wh, nil
	}
	return nil, errors.New("channels: invalid webhook")
}
