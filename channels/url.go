package channels

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	internalauth "github.com/dylanlyu/pusher-go/internal/auth"
)

const authVersion = "1.0"

func buildRequestURL(method, host, path, key, secret string, secure bool, body []byte, queryParams map[string]string, cluster string) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	params := url.Values{
		"auth_key":       {key},
		"auth_timestamp": {timestamp},
		"auth_version":   {authVersion},
	}
	if len(body) > 0 {
		params.Set("body_md5", internalauth.MD5Hex(body))
	}
	for k, v := range queryParams {
		params.Set(k, v)
	}

	// Pusher signs the unescaped query string. params.Encode() always produces
	// valid percent-encoding, so QueryUnescape error is unreachable in practice.
	unescaped, err := url.QueryUnescape(params.Encode())
	if err != nil {
		return "", fmt.Errorf("channels: unescape query string for signing: %w", err)
	}
	stringToSign := strings.Join([]string{method, path, unescaped}, "\n")
	params.Set("auth_signature", internalauth.HMACSignature(stringToSign, secret))

	if host == "" {
		if cluster != "" {
			host = "api-" + cluster + ".pusher.com"
		} else {
			host = "api.pusherapp.com"
		}
	}

	scheme := "http"
	if secure {
		scheme = "https"
	}

	endpoint, err := url.ParseRequestURI(scheme + "://" + host + path)
	if err != nil {
		return "", err
	}
	// Pusher requires unescaped query string in the final URL.
	// params.Encode() always produces valid percent-encoding, so this never errors.
	rawQuery, err := url.QueryUnescape(params.Encode())
	if err != nil {
		return "", fmt.Errorf("channels: unescape query string for URL: %w", err)
	}
	endpoint.RawQuery = rawQuery

	return endpoint.String(), nil
}
