package channels

import (
	"encoding/json"
	"fmt"
)

const defaultMaxPayloadKB = 10

type batchEventPayload struct {
	Channel  string  `json:"channel"`
	Name     string  `json:"name"`
	Data     string  `json:"data"`
	SocketID *string `json:"socket_id,omitempty"`
	Info     *string `json:"info,omitempty"`
}

type batchPayload struct {
	Batch []batchEventPayload `json:"batch"`
}

func encodeTriggerBody(
	channels []string,
	eventName string,
	data any,
	params map[string]string,
	masterKey []byte,
	overrideMaxKB int,
) ([]byte, error) {
	dataBytes, err := encodeEventData(data)
	if err != nil {
		return nil, err
	}

	var payloadData string
	if isEncryptedChannel(channels[0]) {
		payloadData, err = encryptData(channels[0], dataBytes, masterKey)
		if err != nil {
			return nil, err
		}
	} else {
		payloadData = string(dataBytes)
	}

	if err := checkPayloadSize(payloadData, overrideMaxKB); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name":     eventName,
		"channels": channels,
		"data":     payloadData,
	}
	for k, v := range params {
		if _, exists := body[k]; exists {
			return nil, fmt.Errorf("channels: parameter %q specified multiple times", k)
		}
		body[k] = v
	}
	return json.Marshal(body)
}

func encodeTriggerBatchBody(batch []Event, masterKey []byte, overrideMaxKB int) ([]byte, error) {
	events := make([]batchEventPayload, len(batch))
	for i, e := range batch {
		dataBytes, err := encodeEventData(e.Data)
		if err != nil {
			return nil, err
		}
		var payloadData string
		if isEncryptedChannel(e.Channel) {
			payloadData, err = encryptData(e.Channel, dataBytes, masterKey)
			if err != nil {
				return nil, err
			}
		} else {
			payloadData = string(dataBytes)
		}
		if err := checkPayloadSize(payloadData, overrideMaxKB); err != nil {
			return nil, fmt.Errorf("channels: event #%d payload too large: %w", i, err)
		}
		events[i] = batchEventPayload{
			Channel:  e.Channel,
			Name:     e.Name,
			Data:     payloadData,
			SocketID: e.SocketID,
			Info:     e.Info,
		}
	}
	return json.Marshal(&batchPayload{Batch: events})
}

func encodeEventData(data any) ([]byte, error) {
	switch d := data.(type) {
	case []byte:
		return d, nil
	case string:
		return []byte(d), nil
	case nil:
		return []byte("null"), nil
	default:
		return json.Marshal(d)
	}
}

func checkPayloadSize(payload string, overrideMaxKB int) error {
	maxKB := defaultMaxPayloadKB
	if overrideMaxKB > 0 {
		maxKB = overrideMaxKB
	}
	if len(payload) > maxKB*1024 {
		return fmt.Errorf("channels: event payload too large (%d bytes)", len(payload))
	}
	return nil
}
