package fetcher

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
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

type Error struct {
	StatusCode  int
	Description string
	Err         error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Description
	}

	return fmt.Sprintf("%s: %v", e.Description, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
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
func (f *Fetcher) Fetch(url string) (*Result, error) {
	normalizedURL, err := normalizeURL(url)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Get(normalizedURL)
	if err != nil {
		return nil, classifyFetchError(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
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
		return "", &Error{
			StatusCode:  http.StatusBadRequest,
			Description: "please provide a URL to analyze",
		}
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsedURL, err := neturl.ParseRequestURI(trimmed)
	if err != nil || parsedURL.Host == "" {
		return "", &Error{
			StatusCode:  http.StatusBadRequest,
			Description: "the URL format is invalid",
			Err:         err,
		}
	}

	return parsedURL.String(), nil
}

func classifyFetchError(err error) error {
	var urlErr *neturl.Error
	if errors.As(err, &urlErr) {
		var dnsErr *net.DNSError
		if errors.As(urlErr.Err, &dnsErr) {
			return &Error{
				StatusCode:  http.StatusBadRequest,
				Description: "could not find that website",
				Err:         err,
			}
		}

		if urlErr.Timeout() {
			return &Error{
				StatusCode:  http.StatusGatewayTimeout,
				Description: "the website took too long to respond",
				Err:         err,
			}
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &Error{
			StatusCode:  http.StatusGatewayTimeout,
			Description: "the website took too long to respond",
			Err:         err,
		}
	}

	return &Error{
		StatusCode:  http.StatusBadRequest,
		Description: "failed to fetch the webpage",
		Err:         err,
	}
}
