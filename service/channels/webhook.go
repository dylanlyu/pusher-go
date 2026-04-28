package channels

import (
	"encoding/json"
	"fmt"
)

func parseWebhook(body []byte) (*Webhook, error) {
	var wh Webhook
	if err := json.Unmarshal(body, &wh); err != nil {
		return nil, fmt.Errorf("channels: parse webhook body: %w", err)
	}
	return &wh, nil
}
