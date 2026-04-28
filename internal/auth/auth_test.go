package auth_test

import (
	"testing"

	"github.com/dylanlyu/pusher-go/internal/auth"
)

func TestHMACSignature(t *testing.T) {
	tests := []struct {
		name   string
		toSign string
		secret string
		want   string
	}{
		{
			// Precomputed: echo -n "POST\n/apps/..." | openssl dgst -sha256 -hmac "secret"
			name:   "known vector",
			toSign: "POST\n/apps/123/events\nauth_key=key&auth_timestamp=1000&auth_version=1.0&body_md5=abc",
			secret: "secret",
			want:   "d3992f35f239daa8ed2c6572ec321ed94f8b9952f7748bc3a26470f44632fde0",
		},
		{
			// Precomputed: echo -n "" | openssl dgst -sha256 -hmac "secret"
			name:   "empty input",
			toSign: "",
			secret: "secret",
			want:   "f9e66e179b6747ae54108f82f8ade8b3c25d76fd30afde6c395822c530196169",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := auth.HMACSignature(tc.toSign, tc.secret)
			if got != tc.want {
				t.Errorf("HMACSignature() = %q, want %q", got, tc.want)
			}
			if len(got) != 64 {
				t.Errorf("HMACSignature() length = %d, want 64 (hex SHA256)", len(got))
			}
		})
	}
}

func TestCheckSignature(t *testing.T) {
	body := []byte(`{"events":[]}`)
	secret := "my-secret"
	validSig := auth.HMACSignature(string(body), secret)

	tests := []struct {
		name   string
		sig    string
		body   []byte
		secret string
		want   bool
	}{
		{"valid signature", validSig, body, secret, true},
		{"tampered body", validSig, []byte(`{"events":[{"x":1}]}`), secret, false},
		{"wrong secret", validSig, body, "wrong-secret", false},
		{"invalid hex", "not-hex", body, secret, false},
		{"empty signature", "", body, secret, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := auth.CheckSignature(tc.sig, tc.secret, tc.body)
			if got != tc.want {
				t.Errorf("CheckSignature() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCreateAuthMap(t *testing.T) {
	t.Run("without shared secret", func(t *testing.T) {
		m := auth.CreateAuthMap("key", "secret", "string-to-sign", "")
		if _, ok := m["auth"]; !ok {
			t.Error("expected 'auth' key in map")
		}
		if _, ok := m["shared_secret"]; ok {
			t.Error("unexpected 'shared_secret' key in map")
		}
		if len(m) != 1 {
			t.Errorf("map length = %d, want 1", len(m))
		}
	})

	t.Run("with shared secret", func(t *testing.T) {
		m := auth.CreateAuthMap("key", "secret", "string-to-sign", "base64secret==")
		if _, ok := m["auth"]; !ok {
			t.Error("expected 'auth' key in map")
		}
		if _, ok := m["shared_secret"]; !ok {
			t.Error("expected 'shared_secret' key in map")
		}
		if m["shared_secret"] != "base64secret==" {
			t.Errorf("shared_secret = %q, want %q", m["shared_secret"], "base64secret==")
		}
	})

	t.Run("auth value format is key:signature", func(t *testing.T) {
		m := auth.CreateAuthMap("mykey", "mysecret", "data", "")
		expected := "mykey:" + auth.HMACSignature("data", "mysecret")
		if m["auth"] != expected {
			t.Errorf("auth = %q, want %q", m["auth"], expected)
		}
	})
}

func TestMD5Hex(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "known md5",
			body: []byte("hello"),
			want: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name: "empty body",
			body: []byte{},
			want: "d41d8cd98f00b204e9800998ecf8427e",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := auth.MD5Hex(tc.body)
			if got != tc.want {
				t.Errorf("MD5Hex() = %q, want %q", got, tc.want)
			}
		})
	}
}
