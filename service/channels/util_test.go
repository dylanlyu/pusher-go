package channels_test

import (
	"testing"

	"github.com/dylanlyu/pusher-go/service/channels"
)

func TestValidChannel(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		want    bool
	}{
		{"simple", "my-channel", true},
		{"with underscore", "my_channel", true},
		{"with numbers", "channel123", true},
		{"presence prefix", "presence-room", true},
		{"private prefix", "private-chat", true},
		{"encrypted prefix", "private-encrypted-secret", true},
		{"with @", "channel@domain", true},
		{"empty", "", false},
		{"with space", "bad channel", false},
		{"with slash", "bad/channel", false},
		{"too long", string(make([]byte, 201)), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := channels.ValidChannel(tc.channel)
			if got != tc.want {
				t.Errorf("ValidChannel(%q) = %v, want %v", tc.channel, got, tc.want)
			}
		})
	}
}
