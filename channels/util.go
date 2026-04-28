package channels

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

const maxChannelName = 200

var (
	channelNameRegex = regexp.MustCompile(`^[-a-zA-Z0-9_=@,.;]+$`)
	socketIDRegex    = regexp.MustCompile(`\A\d+\.\d+\z`)
)

// ValidChannel reports whether name is a valid Pusher channel name.
// Exported so consumers can validate channel names before creating a client.
func ValidChannel(name string) bool {
	if len(name) == 0 || len(name) > maxChannelName {
		return false
	}
	return channelNameRegex.MatchString(name)
}

func channelsAreValid(channels []string) bool {
	for _, ch := range channels {
		if !ValidChannel(ch) {
			return false
		}
	}
	return true
}

func isEncryptedChannel(channel string) bool {
	return len(channel) >= 18 && channel[:18] == "private-encrypted-"
}

func validateSocketID(socketID *string) error {
	if socketID == nil || socketIDRegex.MatchString(*socketID) {
		return nil
	}
	return errors.New("channels: socket_id is invalid")
}

func validUserID(userID string) bool {
	l := len(userID)
	return l > 0 && l <= maxChannelName
}

func validateUserData(userData map[string]any) error {
	raw, ok := userData["id"]
	if !ok || raw == nil {
		return errors.New("channels: user data is missing id field")
	}
	id, ok := raw.(string)
	if !ok {
		return errors.New("channels: id field in user data must be a string")
	}
	if !validUserID(id) {
		return fmt.Errorf("channels: invalid id in user data: %q", id)
	}
	return nil
}

func parseChannelAuthParams(params []byte) (channelName, socketID string, err error) {
	vals, err := url.ParseQuery(string(params))
	if err != nil {
		return
	}
	if _, ok := vals["channel_name"]; !ok {
		return "", "", errors.New("channels: channel_name not found in params")
	}
	if _, ok := vals["socket_id"]; !ok {
		return "", "", errors.New("channels: socket_id not found in params")
	}
	return vals.Get("channel_name"), vals.Get("socket_id"), nil
}

func parseUserAuthParams(params []byte) (socketID string, err error) {
	vals, err := url.ParseQuery(string(params))
	if err != nil {
		return
	}
	if _, ok := vals["socket_id"]; !ok {
		return "", errors.New("channels: socket_id not found in params")
	}
	return vals.Get("socket_id"), nil
}
