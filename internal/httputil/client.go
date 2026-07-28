// Package httputil provides HTTP helpers with retry and backoff.
package httputil

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// DefaultUserAgent is a browser-like User-Agent for scraping.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Client wraps http.Client with retry logic.
type Client struct {
	HTTPClient  *http.Client
	UserAgent   string
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Accepts     string
	AcceptLang  string
}

// NewDefaultClient returns a Client with sensible defaults for scraping.
func NewDefaultClient() *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent:  DefaultUserAgent,
		MaxRetries: 6,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   8 * time.Second,
		Accepts:    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		AcceptLang: "en-US,en;q=0.9",
	}
}

// FetchHTML fetches a URL and returns the response body as a string.
// Retries on 429 and 5xx with exponential backoff + jitter.
// Returns empty string on 404 (not an error).
func (c *Client) FetchHTML(ctx context.Context, url string) (string, error) {
	delay := c.BaseDelay
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("User-Agent", c.UserAgent)
		req.Header.Set("Accept", c.Accepts)
		req.Header.Set("Accept-Language", c.AcceptLang)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			if attempt == c.MaxRetries {
				return "", fmt.Errorf("request failed after %d retries: %w", c.MaxRetries, err)
			}
			time.Sleep(delay)
			delay = min(delay*2, c.MaxDelay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt == c.MaxRetries {
				return "", fmt.Errorf("request failed: HTTP %d after %d retries", resp.StatusCode, c.MaxRetries)
			}
			jitter := time.Duration(rand.Intn(500)) * time.Millisecond
			time.Sleep(delay + jitter)
			delay = min(delay*2, c.MaxDelay)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			return "", nil
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("request failed: HTTP %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("read body: %w", err)
		}
		return string(body), nil
	}
	return "", fmt.Errorf("request failed after max retries")
}
