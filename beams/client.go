package beams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"

	"github.com/dylanlyu/pusher-go/internal/request"
)

const (
	maxInterests       = 100
	maxInterestNameLen = 164
	maxUserIDLen       = 164
	maxUsersPerPublish = 1000
	tokenTTL           = 24 * time.Hour
	libraryName        = "pusher-go-beams"
	libraryVersion     = "0.1.0"
)

var interestRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-=@,.;]+$`)

func defaultHeaders() map[string]string {
	return map[string]string{
		"Content-Type":     "application/json",
		"X-Pusher-Library": libraryName + " " + libraryVersion,
	}
}

// Client is the Pusher Beams push notification API client interface.
type Client interface {
	PublishToInterests(ctx context.Context, interests []string, body map[string]any) (publishID string, err error)
	PublishToUsers(ctx context.Context, users []string, body map[string]any) (publishID string, err error)
	GenerateToken(userID string) (map[string]any, error)
	DeleteUser(ctx context.Context, userID string) error
}

type client struct {
	cfg beamConfig
}

// New constructs a Client. instanceID and secretKey are required.
func New(instanceID, secretKey string, opts ...Option) (Client, error) {
	if instanceID == "" {
		return nil, errors.New("beams: instanceID is required")
	}
	if secretKey == "" {
		return nil, errors.New("beams: secretKey is required")
	}
	cfg := beamConfig{instanceID: instanceID, secretKey: secretKey}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &client{cfg: cfg}, nil
}

func (c *client) PublishToInterests(ctx context.Context, interests []string, body map[string]any) (string, error) {
	if len(interests) == 0 {
		return "", errors.New("beams: at least one interest is required")
	}
	if len(interests) > maxInterests {
		return "", fmt.Errorf("beams: too many interests supplied (%d), maximum is %d", len(interests), maxInterests)
	}
	for _, interest := range interests {
		if err := validateInterest(interest); err != nil {
			return "", err
		}
	}

	payload := copyMapWithKey(body, "interests", interests)
	return c.publish(ctx, fmt.Sprintf("%s/publish_api/v1/instances/%s/publishes", c.cfg.resolvedBaseURL(), c.cfg.instanceID), payload)
}

func (c *client) PublishToUsers(ctx context.Context, users []string, body map[string]any) (string, error) {
	if len(users) == 0 {
		return "", errors.New("beams: at least one user ID is required")
	}
	if len(users) > maxUsersPerPublish {
		return "", fmt.Errorf("beams: too many user IDs supplied (%d), maximum is %d", len(users), maxUsersPerPublish)
	}
	for i, userID := range users {
		if userID == "" {
			return "", fmt.Errorf("beams: empty user ID at index %d", i)
		}
		if len(userID) > maxUserIDLen {
			return "", fmt.Errorf("beams: user ID at index %d is too long (%d chars, max %d)", i, len(userID), maxUserIDLen)
		}
		if !utf8.ValidString(userID) {
			return "", fmt.Errorf("beams: user ID at index %d is not valid UTF-8", i)
		}
	}

	payload := copyMapWithKey(body, "users", users)
	return c.publish(ctx, fmt.Sprintf("%s/publish_api/v1/instances/%s/publishes/users", c.cfg.resolvedBaseURL(), c.cfg.instanceID), payload)
}

func (c *client) publish(ctx context.Context, endpoint string, payload map[string]any) (string, error) {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("beams: marshal publish request: %w", err)
	}

	headers := defaultHeaders()
	headers["Authorization"] = "Bearer " + c.cfg.secretKey

	respBody, err := request.Do(ctx, c.cfg.httpClientOrDefault(), http.MethodPost, endpoint, reqBody, headers)
	if err != nil {
		return "", fmt.Errorf("beams: publish request failed: %w", err)
	}

	var resp publishResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("beams: parse publish response: %w", err)
	}
	return resp.PublishID, nil
}

func (c *client) GenerateToken(userID string) (map[string]any, error) {
	if userID == "" {
		return nil, errors.New("beams: userID is required")
	}
	if len(userID) > maxUserIDLen {
		return nil, fmt.Errorf("beams: userID is too long (%d chars, max %d)", len(userID), maxUserIDLen)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": jwt.NewNumericDate(time.Now().Add(tokenTTL)),
		"iss": "https://" + c.cfg.instanceID + ".pushnotifications.pusher.com",
	})

	tokenString, err := token.SignedString([]byte(c.cfg.secretKey))
	if err != nil {
		return nil, fmt.Errorf("beams: sign JWT: %w", err)
	}
	return map[string]any{"token": tokenString}, nil
}

func (c *client) DeleteUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("beams: userID is required")
	}
	if len(userID) > maxUserIDLen {
		return fmt.Errorf("beams: userID is too long (%d chars, max %d)", len(userID), maxUserIDLen)
	}
	if !utf8.ValidString(userID) {
		return errors.New("beams: userID must be valid UTF-8")
	}

	endpoint := fmt.Sprintf("%s/customer_api/v1/instances/%s/users/%s",
		c.cfg.resolvedBaseURL(), c.cfg.instanceID, url.PathEscape(userID),
	)
	headers := map[string]string{
		"Content-Type":     "application/json",
		"Authorization":    "Bearer " + c.cfg.secretKey,
		"X-Pusher-Library": libraryName + " " + libraryVersion,
	}

	if _, err := request.Do(ctx, c.cfg.httpClientOrDefault(), http.MethodDelete, endpoint, nil, headers); err != nil {
		// Parse the error response for a better message.
		var errResp errorResponse
		if jsonErr := json.Unmarshal(getErrBody(err), &errResp); jsonErr == nil && errResp.Error != "" {
			return fmt.Errorf("beams: delete user failed: %s: %s", errResp.Error, errResp.Description)
		}
		return fmt.Errorf("beams: delete user failed: %w", err)
	}
	return nil
}

// --- helpers ---

func validateInterest(interest string) error {
	if interest == "" {
		return errors.New("beams: empty interest name is not valid")
	}
	if len(interest) > maxInterestNameLen {
		return fmt.Errorf("beams: interest %q is too long (%d chars, max %d)", interest, len(interest), maxInterestNameLen)
	}
	if !interestRegex.MatchString(interest) {
		return fmt.Errorf("beams: interest %q contains invalid characters", interest)
	}
	return nil
}

// copyMapWithKey returns a shallow copy of m with key set to value.
func copyMapWithKey(m map[string]any, key string, value any) map[string]any {
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out[key] = value
	return out
}

// getErrBody extracts the body bytes from a request.ErrHTTP if possible.
func getErrBody(err error) []byte {
	var httpErr *request.ErrHTTP
	if errors.As(err, &httpErr) {
		return httpErr.Body
	}
	return nil
}
