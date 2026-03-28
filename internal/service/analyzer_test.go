package service

import (
	"errors"
	"strings"
	"testing"

	"home24se-take-home/internal/fetcher"
	"home24se-take-home/internal/model"
)

/*
TestNew verifies that the constructor keeps the provided dependencies and
returns a usable analyzer instance.
*/
func TestNew(t *testing.T) {
	t.Parallel()

	fetcherMock := &fetcherStub{}
	parserMock := &parserStub{}

	analyzer := New(fetcherMock, parserMock)
	if analyzer == nil {
		t.Fatal("New() returned nil")
	}

	if analyzer.fetcher != fetcherMock {
		t.Fatal("New() did not keep provided fetcher")
	}

	if analyzer.parser != parserMock {
		t.Fatal("New() did not keep provided parser")
	}

	if analyzer.checker == nil {
		t.Fatal("New() did not create default link checker")
	}
}

/*
TestAnalyzeSuccess verifies the full happy path. It checks that the service
fetches the page, passes the fetch result final URL and HTML body into the
parser, and returns the parser output unchanged.
*/
func TestAnalyzeSuccess(t *testing.T) {
	t.Parallel()

	fetcherMock := &fetcherStub{
		result: &fetcher.Result{
			Body:       []byte("<html><head><title>Example</title></head></html>"),
			StatusCode: 200,
			FinalURL:   "https://example.com/final",
		},
	}

	expected := model.Result{
		HTMLVersion:   "HTML5",
		Title:         "Example",
		Headings:      map[string]int{"h1": 1},
		InternalLinks: []string{"/about"},
		ExternalLinks: []string{"https://golang.org"},
		HasLoginForm:  false,
	}

	parserMock := &parserStub{
		result: expected,
	}

	linkCheckerMock := &linkCheckerStub{
		accessible: map[string]bool{
			"https://example.com/about": false,
		},
	}

	analyzer := NewWithChecker(fetcherMock, parserMock, linkCheckerMock)

	result, err := analyzer.Analyze("https://example.com")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if fetcherMock.calledWith != "https://example.com" {
		t.Fatalf("fetcher called with %q, want %q", fetcherMock.calledWith, "https://example.com")
	}

	if parserMock.documentURL != "https://example.com/final" {
		t.Fatalf("parser documentURL = %q, want %q", parserMock.documentURL, "https://example.com/final")
	}

	if parserMock.htmlContent != "<html><head><title>Example</title></head></html>" {
		t.Fatalf("parser htmlContent = %q, want fetched body", parserMock.htmlContent)
	}

	if result.HTMLVersion != expected.HTMLVersion ||
		result.Title != expected.Title ||
		result.HasLoginForm != expected.HasLoginForm {
		t.Fatalf("Analyze() result = %#v, want %#v", result, expected)
	}

	if len(result.InternalLinks) != 1 || len(result.ExternalLinks) != 1 {
		t.Fatalf("Analyze() links = %#v / %#v", result.InternalLinks, result.ExternalLinks)
	}

	if result.BrokenLinks != 1 {
		t.Fatalf("BrokenLinks = %d, want %d", result.BrokenLinks, 1)
	}
}

/*
TestAnalyzeFetchError verifies that a fetch failure stops the workflow before
parsing and that the returned error is wrapped with service-level context.
*/
func TestAnalyzeFetchError(t *testing.T) {
	t.Parallel()

	fetcherMock := &fetcherStub{
		err: errors.New("network down"),
	}
	parserMock := &parserStub{}

	analyzer := NewWithChecker(fetcherMock, parserMock, &linkCheckerStub{})

	_, err := analyzer.Analyze("https://example.com")
	if err == nil {
		t.Fatal("Analyze() error = nil, want fetch error")
	}

	if !strings.Contains(err.Error(), "fetch webpage") {
		t.Fatalf("error = %q, want wrapped fetch error", err.Error())
	}

	if parserMock.called {
		t.Fatal("parser should not be called after fetch failure")
	}
}

/*
TestAnalyzeParserError verifies that parser failures are propagated with
service-level context after a successful fetch.
*/
func TestAnalyzeParserError(t *testing.T) {
	t.Parallel()

	fetcherMock := &fetcherStub{
		result: &fetcher.Result{
			Body:       []byte("<html></html>"),
			StatusCode: 200,
			FinalURL:   "https://example.com/final",
		},
	}
	parserMock := &parserStub{
		err: errors.New("parse failed"),
	}

	analyzer := NewWithChecker(fetcherMock, parserMock, &linkCheckerStub{})

	_, err := analyzer.Analyze("https://example.com")
	if err == nil {
		t.Fatal("Analyze() error = nil, want parser error")
	}

	if !strings.Contains(err.Error(), "parse webpage") {
		t.Fatalf("error = %q, want wrapped parser error", err.Error())
	}
}

/*
TestAnalyzeCountsOnlyBrokenInternalLinks verifies that the service checks the
internal links produced by the parser, resolves them against the analyzed URL,
and increments BrokenLinks only for inaccessible internal targets.
*/
func TestAnalyzeCountsOnlyBrokenInternalLinks(t *testing.T) {
	t.Parallel()

	fetcherMock := &fetcherStub{
		result: &fetcher.Result{
			Body:       []byte("<html></html>"),
			StatusCode: 200,
			FinalURL:   "https://example.com/root/page",
		},
	}

	parserMock := &parserStub{
		result: model.Result{
			InternalLinks: []string{"/missing", "relative"},
			ExternalLinks: []string{"https://golang.org"},
		},
	}

	linkCheckerMock := &linkCheckerStub{
		accessible: map[string]bool{
			"https://example.com/missing":       false,
			"https://example.com/root/relative": true,
		},
	}

	analyzer := NewWithChecker(fetcherMock, parserMock, linkCheckerMock)

	result, err := analyzer.Analyze("https://example.com")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if result.BrokenLinks != 1 {
		t.Fatalf("BrokenLinks = %d, want %d", result.BrokenLinks, 1)
	}
}

type fetcherStub struct {
	result     *fetcher.Result
	err        error
	calledWith string
}

func (f *fetcherStub) Fetch(url string) (*fetcher.Result, error) {
	f.calledWith = url
	if f.err != nil {
		return nil, f.err
	}

	return f.result, nil
}

type parserStub struct {
	result      model.Result
	err         error
	called      bool
	documentURL string
	htmlContent string
}

func (p *parserStub) Parse(documentURL string, htmlContent string) (model.Result, error) {
	p.called = true
	p.documentURL = documentURL
	p.htmlContent = htmlContent

	if p.err != nil {
		return model.Result{}, p.err
	}

	return p.result, nil
}

type linkCheckerStub struct {
	accessible map[string]bool
}

func (l *linkCheckerStub) IsAccessible(link string) bool {
	return l.accessible[link]
}
