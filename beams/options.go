package beams

import "net/http"

// Option configures a beams Client.
type Option func(*beamConfig)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *beamConfig) { c.httpClient = hc }
}

// WithBaseURL overrides the default Beams API endpoint.
func WithBaseURL(url string) Option {
	return func(c *beamConfig) { c.baseURL = url }
}

type beamConfig struct {
	instanceID string
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

func (c *beamConfig) resolvedBaseURL() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return "https://" + c.instanceID + ".pushnotifications.pusher.com"
}

func (c *beamConfig) httpClientOrDefault() *http.Client {
	if c.httpClient != nil {
		return c.httpClient
	}
	return http.DefaultClient
}
