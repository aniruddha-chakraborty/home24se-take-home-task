package fetcher

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

/*
TestNew verifies that the default constructor returns a usable fetcher
with an initialized HTTP client.
*/
func TestNew(t *testing.T) {
	t.Parallel()

	f := New()
	if f == nil {
		t.Fatal("New() returned nil")
	}

	if f.client == nil {
		t.Fatal("New() returned fetcher with nil client")
	}
}

/*
TestNewWithClient verifies that callers can inject their own HTTP client and
that the fetcher uses exactly that client instance.
*/
func TestNewWithClient(t *testing.T) {
	t.Parallel()

	client := &http.Client{}
	f := NewWithClient(client)

	if f.client != client {
		t.Fatal("NewWithClient() did not keep the provided client")
	}
}

/*
TestNewWithClientNil verifies the nil-client fallback path and ensures the
constructor still produces a usable fetcher.
*/
func TestNewWithClientNil(t *testing.T) {
	t.Parallel()

	f := NewWithClient(nil)
	if f == nil {
		t.Fatal("NewWithClient(nil) returned nil")
	}

	if f.client == nil {
		t.Fatal("NewWithClient(nil) returned fetcher with nil client")
	}
}

/*
TestFetchSuccess verifies the happy path: the fetcher downloads the response,
returns the exact body bytes, and exposes the HTTP status code and request URL.
*/
func TestFetchSuccess(t *testing.T) {
	t.Parallel()

	f := NewWithClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("hello world")),
				Request:    req,
				Header:     make(http.Header),
			}, nil
		}),
	})

	result, err := f.Fetch("https://example.com")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if string(result.Body) != "hello world" {
		t.Fatalf("Body = %q, want %q", string(result.Body), "hello world")
	}

	if result.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", result.StatusCode, http.StatusOK)
	}

	if result.FinalURL != "https://example.com" {
		t.Fatalf("FinalURL = %q, want %q", result.FinalURL, "https://example.com")
	}
}

/*
TestFetchRedirect verifies that the fetcher follows redirects via the HTTP
client and reports the final URL after redirection.
*/
func TestFetchRedirect(t *testing.T) {
	t.Parallel()

	f := NewWithClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("redirect target")),
				Request: &http.Request{
					Method: req.Method,
					URL:    mustParseURL("https://example.com/final"),
					Header: make(http.Header),
				},
				Header: make(http.Header),
			}, nil
		}),
	})

	result, err := f.Fetch("https://example.com/start")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if result.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", result.StatusCode, http.StatusOK)
	}

	if result.FinalURL != "https://example.com/final" {
		t.Fatalf("FinalURL = %q, want %q", result.FinalURL, "https://example.com/final")
	}
}

/*
TestFetchNon2xxStatus verifies that non-success status codes are still returned
as results rather than being treated as transport errors.
*/
func TestFetchNon2xxStatus(t *testing.T) {
	t.Parallel()

	f := NewWithClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("upstream issue")),
				Request:    req,
				Header:     make(http.Header),
			}, nil
		}),
	})

	result, err := f.Fetch("https://example.com")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if result.StatusCode != http.StatusBadGateway {
		t.Fatalf("StatusCode = %d, want %d", result.StatusCode, http.StatusBadGateway)
	}

	if !strings.Contains(string(result.Body), "upstream issue") {
		t.Fatalf("Body = %q, want error response body", string(result.Body))
	}
}

/*
TestFetchTransportError verifies the failure path when the HTTP client cannot
reach the target URL at all.
*/
func TestFetchTransportError(t *testing.T) {
	t.Parallel()

	f := NewWithClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: errors.New("no such host"),
			}
		}),
	})

	_, err := f.Fetch("https://example.com")
	if err == nil {
		t.Fatal("Fetch() error = nil, want transport error")
	}

	if !strings.Contains(err.Error(), "failed to fetch URL") {
		t.Fatalf("error = %q, want wrapped fetch error", err.Error())
	}
}

/*
TestFetchReadBodyError verifies the failure path when the response body cannot
be fully read after a successful HTTP response.
*/
func TestFetchReadBodyError(t *testing.T) {
	t.Parallel()

	f := NewWithClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(errReader{}),
				Request:    &http.Request{URL: mustParseURL("https://example.com")},
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, err := f.Fetch("https://example.com")
	if err == nil {
		t.Fatal("Fetch() error = nil, want read error")
	}

	if !strings.Contains(err.Error(), "failed to read response body") {
		t.Fatalf("error = %q, want wrapped body read error", err.Error())
	}
}

/*
TestFetchRejectsInvalidURL verifies that malformed input is rejected with a
clean user-facing error before any HTTP request is attempted.
*/
func TestFetchRejectsInvalidURL(t *testing.T) {
	t.Parallel()

	f := New()

	_, err := f.Fetch("://bad-url")
	if err == nil {
		t.Fatal("Fetch() error = nil, want invalid URL error")
	}

	if !strings.Contains(err.Error(), "the URL format is invalid") {
		t.Fatalf("error = %q, want invalid URL message", err.Error())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func mustParseURL(raw string) *url.URL {
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		panic(err)
	}

	return req.URL
}
