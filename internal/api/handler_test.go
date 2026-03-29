package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"home24se-take-home/internal/model"
)

/*
TestAnalyzeSuccess verifies that the handler accepts a valid JSON request,
calls the analyzer service, and returns the frontend-facing JSON shape.
*/
func TestAnalyzeSuccess(t *testing.T) {
	t.Parallel()

	serviceStub := &analyzerStub{
		result: model.Result{
			AnalyzedURL:   "https://example.com/final",
			HTMLVersion:   "HTML5",
			Title:         "Example",
			Headings:      map[string]int{"h1": 1, "h2": 2},
			InternalLinks: []string{"/a", "/b"},
			ExternalLinks: []string{"https://golang.org"},
			BrokenLinks:   1,
			HasLoginForm:  true,
		},
	}

	handler := NewHandlerWithService(serviceStub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Analyze(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var response analyzeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if serviceStub.calledWith != "https://example.com" {
		t.Fatalf("analyzer called with %q, want %q", serviceStub.calledWith, "https://example.com")
	}

	if response.Result == nil {
		t.Fatal("result is nil")
	}

	if response.Result.AnalyzedURL != "https://example.com/final" {
		t.Fatalf("AnalyzedURL = %q", response.Result.AnalyzedURL)
	}

	if response.Result.InternalLinks != 2 {
		t.Fatalf("InternalLinks = %d, want 2", response.Result.InternalLinks)
	}

	if response.Result.ExternalLinks != 1 {
		t.Fatalf("ExternalLinks = %d, want 1", response.Result.ExternalLinks)
	}

	if !response.Result.HasLoginForm {
		t.Fatal("HasLoginForm = false, want true")
	}
}

/*
TestAnalyzeRejectsInvalidMethod verifies that only POST requests are accepted.
*/
func TestAnalyzeRejectsInvalidMethod(t *testing.T) {
	t.Parallel()

	serviceStub := &analyzerStub{}
	handler := NewHandlerWithService(serviceStub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analyze", nil)
	rr := httptest.NewRecorder()

	handler.Analyze(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}

	if serviceStub.calledWith != "" {
		t.Fatalf("analyzer should not be called, got %q", serviceStub.calledWith)
	}
}

/*
TestAnalyzeRejectsInvalidJSON verifies request validation when the JSON body
cannot be decoded.
*/
func TestAnalyzeRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	serviceStub := &analyzerStub{}
	handler := NewHandlerWithService(serviceStub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(`{"url":`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Analyze(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	if serviceStub.calledWith != "" {
		t.Fatalf("analyzer should not be called, got %q", serviceStub.calledWith)
	}
}

/*
TestAnalyzeRejectsEmptyURL verifies request validation when the client sends
an empty URL.
*/
func TestAnalyzeRejectsEmptyURL(t *testing.T) {
	t.Parallel()

	serviceStub := &analyzerStub{}
	handler := NewHandlerWithService(serviceStub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(`{"url":""}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Analyze(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	if serviceStub.calledWith != "" {
		t.Fatalf("analyzer should not be called, got %q", serviceStub.calledWith)
	}
}

/*
TestAnalyzeServiceError verifies that service-level failures are returned as a
frontend-readable error response.
*/
func TestAnalyzeServiceError(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithService(&analyzerStub{
		err: errors.New("fetch webpage: failed to fetch URL"),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Analyze(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var response analyzeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response.Error == nil {
		t.Fatal("error response is nil")
	}
}

/*
TestAnalyzeMapsFriendlyFetchError verifies that transport-looking errors are
turned into cleaner frontend-facing messages.
*/
func TestAnalyzeMapsFriendlyFetchError(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithService(&analyzerStub{
		err: errors.New(`fetch webpage: failed to fetch URL: Get "https://example.com": dial tcp: lookup example.com: no such host`),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Analyze(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var response analyzeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response.Error == nil {
		t.Fatal("error response is nil")
	}

	if response.Error.Description != "could not find that website" {
		t.Fatalf("Description = %q, want %q", response.Error.Description, "could not find that website")
	}
}

/*
TestAnalyzeMapsUpstreamHTTPStatus verifies that target-site HTTP failures are
returned with their upstream status code instead of being parsed as page data.
*/
func TestAnalyzeMapsUpstreamHTTPStatus(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithService(&analyzerStub{
		err: errors.New("fetch webpage: target website returned HTTP status 403"),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Analyze(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	var response analyzeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response.Error == nil {
		t.Fatal("error response is nil")
	}

	if response.Error.Description != "the website blocked the request" {
		t.Fatalf("Description = %q, want %q", response.Error.Description, "the website blocked the request")
	}
}

type analyzerStub struct {
	result     model.Result
	err        error
	calledWith string
}

func (a *analyzerStub) Analyze(url string) (model.Result, error) {
	a.calledWith = url
	if a.err != nil {
		return model.Result{}, a.err
	}
	return a.result, nil
}
