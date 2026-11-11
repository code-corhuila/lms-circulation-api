// Package iamclient calls lms-iam's GET /api/v1/iam/authorize
// (ADR-012-iam-service-scope.md) — the one decision endpoint this service
// is meant to check before a protected write, closing T2/SEC-04 from the
// week-13 security threat model.
package iamclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Client struct {
	baseURL   string
	jwtSecret string
	http      *http.Client
}

func NewClient(baseURL, jwtSecret string) *Client {
	return &Client{baseURL: baseURL, jwtSecret: jwtSecret, http: &http.Client{Timeout: 5 * time.Second}}
}

type authorizeResponse struct {
	Allowed bool `json:"allowed"`
}

// Authorize asks lms-iam whether administratorID's role grants permission.
// An administrator with no role assignment defaults to allowed=true
// (ADR-012's backward-compatible default) — this client has no fallback
// logic of its own, it only reports what lms-iam answers.
func (c *Client) Authorize(ctx context.Context, administratorID, permission string) (bool, error) {
	token, err := c.mintServiceToken()
	if err != nil {
		return false, fmt.Errorf("minting internal service token: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/iam/authorize?administratorId=%s&permission=%s",
		c.baseURL, url.QueryEscape(administratorID), url.QueryEscape(permission))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("calling lms-iam: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("lms-iam returned %d", resp.StatusCode)
	}

	var body authorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, fmt.Errorf("decoding lms-iam response: %w", err)
	}
	return body.Allowed, nil
}

// mintServiceToken signs a short-lived internal token with
// INTERNAL_JWT_SECRET — deliberately not any end-user RS256 session token
// (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.7).
func (c *Client) mintServiceToken() (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub": "circulation-service",
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.jwtSecret))
}
