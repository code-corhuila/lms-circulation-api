// Package catalog implements the service.BookClient driven port over HTTP —
// LoanRegistrationService can no longer call catalog's BookRepository
// directly now that Book lives in a different database
// (library-docs/09-microservices/service-boundary-rules.md).
package catalog

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrBookNotFound and ErrNoCopiesAvailable mirror catalog-service's 404/409
// for /books/{id}/loan-copy and /return-copy.
var (
	ErrBookNotFound      = fmt.Errorf("book not found")
	ErrNoCopiesAvailable = fmt.Errorf("no copies available")
)

type Client struct {
	baseURL   string
	jwtSecret string
	http      *http.Client
}

func NewClient(baseURL, jwtSecret string) *Client {
	return &Client{baseURL: baseURL, jwtSecret: jwtSecret, http: &http.Client{Timeout: 5 * time.Second}}
}

// LoanCopy asks catalog-service to decrement a book's available copies —
// the mirror of catalog.Book.LoanOneCopy(), now over the network.
func (c *Client) LoanCopy(ctx context.Context, bookID string) error {
	resp, err := c.post(ctx, fmt.Sprintf("/api/v1/books/%s/loan-copy", bookID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrBookNotFound
	case http.StatusConflict:
		return ErrNoCopiesAvailable
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("catalog-service returned %d", resp.StatusCode)
	}
}

// ReturnCopy asks catalog-service to increment a book's available copies —
// the mirror of catalog.Book.ReturnOneCopy().
func (c *Client) ReturnCopy(ctx context.Context, bookID string) error {
	resp, err := c.post(ctx, fmt.Sprintf("/api/v1/books/%s/return-copy", bookID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrBookNotFound
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("catalog-service returned %d", resp.StatusCode)
	}
}

func (c *Client) post(ctx context.Context, path string) (*http.Response, error) {
	token, err := c.mintServiceToken()
	if err != nil {
		return nil, fmt.Errorf("minting internal service token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling catalog-service: %w", err)
	}
	return resp, nil
}

// mintServiceToken signs a short-lived internal token with the JWT_SECRET
// shared by every service — v1 has no separate service-to-service auth
// scope, an accepted trade-off at this size.
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
