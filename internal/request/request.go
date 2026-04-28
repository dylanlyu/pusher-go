package request

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// ErrHTTP is returned when the server responds with a non-2xx status code.
type ErrHTTP struct {
	StatusCode int
	Body       []byte
}

func (e *ErrHTTP) Error() string {
	return fmt.Sprintf("pusher: status %d: %s", e.StatusCode, e.Body)
}

// Do executes an HTTP request and returns the response body.
// A non-2xx status code is returned as an *ErrHTTP error.
func Do(ctx context.Context, client *http.Client, method, url string, body []byte, extraHeaders map[string]string) ([]byte, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("pusher: create request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pusher: execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pusher: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ErrHTTP{StatusCode: resp.StatusCode, Body: respBody}
	}

	return respBody, nil
}
