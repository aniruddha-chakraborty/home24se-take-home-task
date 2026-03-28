package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

type Fetcher struct {
	client *http.Client
}

type Result struct {
	Body       []byte
	StatusCode int
	FinalURL   string
}

// New creates a fetcher with the default HTTP client timeout.
func New() *Fetcher {
	return NewWithClient(&http.Client{
		Timeout: defaultTimeout,
	})
}

// NewWithClient lets tests or callers inject a custom HTTP client.
func NewWithClient(client *http.Client) *Fetcher {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	return &Fetcher{
		client: client,
	}
}

// Fetch downloads the given URL, follows redirects supported by the client,
// and returns the response body together with status code and final URL.
func (f *Fetcher) Fetch(rawURL string) (*Result, error) {
	normalizedURL, err := normalizeURL(rawURL)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Get(normalizedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("target website returned HTTP status %d", resp.StatusCode)
	}

	return &Result{
		Body:       body,
		StatusCode: resp.StatusCode,
		FinalURL:   resp.Request.URL.String(),
	}, nil
}

func normalizeURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("please provide a URL to analyze")
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsedURL, err := url.ParseRequestURI(trimmed)
	if err != nil || parsedURL.Host == "" {
		return "", fmt.Errorf("the URL format is invalid")
	}

	return parsedURL.String(), nil
}
