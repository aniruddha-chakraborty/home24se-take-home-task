package service

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"home24se-take-home/internal/fetcher"
	"home24se-take-home/internal/model"
)

type fetcherClient interface {
	Fetch(url string) (*fetcher.Result, error)
}

type parserClient interface {
	Parse(documentURL string, htmlContent string) (model.Result, error)
}

type linkChecker interface {
	IsAccessible(link string) bool
}

type Analyzer struct {
	fetcher fetcherClient
	parser  parserClient
	checker linkChecker
}

type HTTPLinkChecker struct {
	client *http.Client
}

// New creates the analyzer service with the provided dependencies.
func New(fetcher fetcherClient, parser parserClient) *Analyzer {
	return NewWithChecker(fetcher, parser, NewHTTPLinkChecker())
}

// NewWithChecker creates the analyzer service with a custom link checker.
func NewWithChecker(fetcher fetcherClient, parser parserClient, checker linkChecker) *Analyzer {
	return &Analyzer{
		fetcher: fetcher,
		parser:  parser,
		checker: checker,
	}
}

func NewHTTPLinkChecker() *HTTPLinkChecker {
	return &HTTPLinkChecker{
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Analyze coordinates webpage fetching and HTML parsing into one result.
func (a *Analyzer) Analyze(url string) (model.Result, error) {
	fetchResult, err := a.fetcher.Fetch(url)
	if err != nil {
		return model.Result{}, fmt.Errorf("fetch webpage: %w", err)
	}

	result, err := a.parser.Parse(fetchResult.FinalURL, string(fetchResult.Body))
	if err != nil {
		return model.Result{}, fmt.Errorf("parse webpage: %w", err)
	}

	result.AnalyzedURL = fetchResult.FinalURL
	result.BrokenLinks = a.countBrokenInternalLinks(result.AnalyzedURL, result.InternalLinks)

	return result, nil
}

func (a *Analyzer) countBrokenInternalLinks(documentURL string, links []string) int {
	if a.checker == nil || len(links) == 0 {
		return 0
	}

	baseURL, err := url.Parse(documentURL)
	if err != nil {
		return 0
	}

	var (
		brokenCount int
		mu          sync.Mutex
		wg          sync.WaitGroup
	)

	for _, link := range links {
		link := link
		wg.Add(1)

		go func() {
			defer wg.Done()

			resolved := baseURL.ResolveReference(&url.URL{Path: link})
			if parsed, err := url.Parse(link); err == nil {
				resolved = baseURL.ResolveReference(parsed)
			}

			if a.checker.IsAccessible(resolved.String()) {
				return
			}

			mu.Lock()
			brokenCount++
			mu.Unlock()
		}()
	}

	wg.Wait()

	return brokenCount
}

func (c *HTTPLinkChecker) IsAccessible(link string) bool {
	req, err := http.NewRequest(http.MethodHead, link, nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err == nil {
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode < http.StatusBadRequest {
			return true
		}
	}

	req, err = http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return false
	}

	resp, err = c.client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return resp.StatusCode < http.StatusBadRequest
}
