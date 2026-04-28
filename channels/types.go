package channels

// TriggerParams holds optional parameters for trigger requests.
type TriggerParams struct {
	SocketID *string
	Info     *string
}

func (p TriggerParams) toMap() map[string]string {
	m := make(map[string]string)
	if p.SocketID != nil {
		m["socket_id"] = *p.SocketID
	}
	if p.Info != nil {
		m["info"] = *p.Info
	}
	return m
}

// Event holds data for a single event in a batch trigger request.
type Event struct {
	Channel  string
	Name     string
	Data     any
	SocketID *string
	Info     *string
}

// ChannelsParams holds optional parameters for listing channels.
type ChannelsParams struct {
	FilterByPrefix *string
	Info           *string
}

func (p ChannelsParams) toMap() map[string]string {
	m := make(map[string]string)
	if p.FilterByPrefix != nil {
		m["filter_by_prefix"] = *p.FilterByPrefix
	}
	if p.Info != nil {
		m["info"] = *p.Info
	}
	return m
}

// ChannelParams holds optional parameters for a single channel query.
type ChannelParams struct {
	Info *string
}

func (p ChannelParams) toMap() map[string]string {
	m := make(map[string]string)
	if p.Info != nil {
		m["info"] = *p.Info
	}
	return m
}

// Channel represents information about a single Pusher channel.
type Channel struct {
	Name              string
	Occupied          bool `json:"occupied,omitempty"`
	UserCount         int  `json:"user_count,omitempty"`
	SubscriptionCount int  `json:"subscription_count,omitempty"`
}

// ChannelsList represents a list of channels from the Pusher API.
type ChannelsList struct {
	Channels map[string]ChannelListItem `json:"channels"`
}

// ChannelListItem is an element within ChannelsList.
type ChannelListItem struct {
	UserCount int `json:"user_count"`
}

// TriggerChannelsList is returned by TriggerWithParams when Info is requested.
type TriggerChannelsList struct {
	Channels map[string]TriggerChannelListItem `json:"channels"`
}

// TriggerChannelListItem holds per-channel info returned by a trigger.
type TriggerChannelListItem struct {
	UserCount         *int `json:"user_count,omitempty"`
	SubscriptionCount *int `json:"subscription_count,omitempty"`
}

// TriggerBatchChannelsList is returned by TriggerBatch.
type TriggerBatchChannelsList struct {
	Batch []TriggerBatchChannelListItem `json:"batch"`
}

// TriggerBatchChannelListItem holds per-event info from a batch trigger.
type TriggerBatchChannelListItem struct {
	UserCount         *int `json:"user_count,omitempty"`
	SubscriptionCount *int `json:"subscription_count,omitempty"`
}

// Users represents a list of users in a presence channel.
type Users struct {
	List []User `json:"users"`
}

// User represents a user in a presence channel.
type User struct {
	ID string `json:"id"`
}

// MemberData is passed when authorizing a presence channel subscription.
type MemberData struct {
	UserID   string            `json:"user_id"`
	UserInfo map[string]string `json:"user_info,omitempty"`
}

// Webhook is the parsed form of a valid webhook payload.
type Webhook struct {
	TimeMs int            `json:"time_ms"`
	Events []WebhookEvent `json:"events"`
}

// WebhookEvent represents a single event in a webhook payload.
type WebhookEvent struct {
	Name     string `json:"name"`
	Channel  string `json:"channel"`
	Event    string `json:"event,omitempty"`
	Data     string `json:"data,omitempty"`
	SocketID string `json:"socket_id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
}

// EncryptedMessage holds NaCl-encrypted event data.
type EncryptedMessage struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}
