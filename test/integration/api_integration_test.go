package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type analyzeRequest struct {
	URL string `json:"url"`
}

type analyzeResponse struct {
	Result *struct {
		AnalyzedURL   string         `json:"analyzedURL"`
		HTMLVersion   string         `json:"htmlVersion"`
		Title         string         `json:"title"`
		Headings      map[string]int `json:"headings"`
		InternalLinks int            `json:"internalLinks"`
		ExternalLinks int            `json:"externalLinks"`
		BrokenLinks   int            `json:"brokenLinks"`
		HasLoginForm  bool           `json:"hasLoginForm"`
	} `json:"result,omitempty"`
	Error *struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"error,omitempty"`
}

type fixtureExpectation struct {
	file          string
	htmlVersion   string
	title         string
	headings      map[string]int
	internalLinks int
	externalLinks int
	brokenLinks   int
	hasLoginForm  bool
}

/*
TestIntegrationAnalyzeFixtures validates the live HTTP API end-to-end against
the HTML fixtures served by the companion web server. It performs a simple
readiness check against the analyzer service first, then runs the fixture
subtests against the analyze endpoint.
*/
func TestIntegrationAnalyzeFixtures(t *testing.T) {
	baseURL := analyzerBaseURL()
	fixtureBaseURL := fixtureBaseURL()

	assertHealthOK(t, baseURL)

	tests := []fixtureExpectation{
		{
			file:        "01_html5_basic.html",
			htmlVersion: "HTML5",
			title:       "HTML5 Basic",
			headings:    map[string]int{"h1": 1},
		},
		{
			file:        "02_html401.html",
			htmlVersion: "HTML 4.01",
			title:       "HTML 4.01 Fixture",
			headings:    map[string]int{"h1": 1},
		},
		{
			file:        "03_xhtml10.html",
			htmlVersion: "XHTML 1.0",
			title:       "XHTML 1.0 Fixture",
			headings:    map[string]int{"h1": 1},
		},
		{
			file:        "04_xhtml11.html",
			htmlVersion: "XHTML 1.1",
			title:       "XHTML 1.1 Fixture",
			headings:    map[string]int{"h1": 1},
		},
		{
			file:        "05_no_doctype.html",
			htmlVersion: "Unknown",
			title:       "No Doctype",
			headings:    map[string]int{"h2": 1},
		},
		{
			file:        "06_no_title.html",
			htmlVersion: "HTML5",
			title:       "",
			headings:    map[string]int{"h1": 1},
		},
		{
			file:        "07_headings_all_levels.html",
			htmlVersion: "HTML5",
			title:       "All Headings",
			headings:    map[string]int{"h1": 1, "h2": 2, "h3": 1, "h4": 1, "h5": 1, "h6": 1},
		},
		{
			file:          "08_internal_relative_links.html",
			htmlVersion:   "HTML5",
			title:         "Relative Links",
			internalLinks: 3,
			externalLinks: 0,
		},
		{
			file:          "09_external_links.html",
			htmlVersion:   "HTML5",
			title:         "External Links",
			internalLinks: 0,
			externalLinks: 3,
		},
		{
			file:          "10_mixed_links.html",
			htmlVersion:   "HTML5",
			title:         "Mixed Links",
			internalLinks: 2,
			externalLinks: 2,
		},
		{
			file:          "11_ignored_links.html",
			htmlVersion:   "HTML5",
			title:         "Ignored Links",
			internalLinks: 0,
			externalLinks: 0,
		},
		{
			file:         "12_login_password.html",
			htmlVersion:  "HTML5",
			title:        "Password Login",
			hasLoginForm: true,
		},
		{
			file:         "13_login_keyword_only.html",
			htmlVersion:  "HTML5",
			title:        "Keyword Login",
			hasLoginForm: true,
		},
		{
			file:         "14_no_login_form.html",
			htmlVersion:  "HTML5",
			title:        "Newsletter Form",
			hasLoginForm: true,
		},
		{
			file:          "15_same_host_absolute_links.html",
			htmlVersion:   "HTML5",
			title:         "Same Host Absolute",
			internalLinks: 0,
			externalLinks: 2,
		},
		{
			file:          "16_subdomain_links.html",
			htmlVersion:   "HTML5",
			title:         "Subdomain Links",
			internalLinks: 0,
			externalLinks: 3,
		},
		{
			file:          "17_broken_internal_links.html",
			htmlVersion:   "HTML5",
			title:         "Broken Internal Links",
			internalLinks: 3,
			externalLinks: 0,
			brokenLinks:   0,
		},
		{
			file:         "18_form_with_multilingual_text.html",
			htmlVersion:  "HTML5",
			title:        "Anmeldung",
			hasLoginForm: true,
		},
		{
			file:          "19_malformed_but_parseable.html",
			htmlVersion:   "HTML5",
			title:         "Broken Layout",
			headings:      map[string]int{"h1": 1},
			internalLinks: 1,
		},
		{
			file:          "20_complex_page.html",
			htmlVersion:   "HTML5",
			title:         "Complex Fixture",
			headings:      map[string]int{"h1": 1, "h2": 2, "h3": 1},
			internalLinks: 2,
			externalLinks: 2,
			hasLoginForm:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.file, func(t *testing.T) {
			fixtureURL := fmt.Sprintf("%s/%s", fixtureBaseURL, tt.file)
			result := analyzeFixture(t, baseURL, fixtureURL)

			if result.Result == nil {
				t.Fatal("result is nil")
			}

			if result.Result.HTMLVersion != tt.htmlVersion {
				t.Fatalf("HTMLVersion = %q, want %q", result.Result.HTMLVersion, tt.htmlVersion)
			}

			if result.Result.Title != tt.title {
				t.Fatalf("Title = %q, want %q", result.Result.Title, tt.title)
			}

			for level, want := range tt.headings {
				got := result.Result.Headings[level]
				if got != want {
					t.Fatalf("Headings[%q] = %d, want %d", level, got, want)
				}
			}

			if result.Result.InternalLinks != tt.internalLinks {
				t.Fatalf("InternalLinks = %d, want %d", result.Result.InternalLinks, tt.internalLinks)
			}

			if result.Result.ExternalLinks != tt.externalLinks {
				t.Fatalf("ExternalLinks = %d, want %d", result.Result.ExternalLinks, tt.externalLinks)
			}

			if result.Result.BrokenLinks != tt.brokenLinks {
				t.Fatalf("BrokenLinks = %d, want %d", result.Result.BrokenLinks, tt.brokenLinks)
			}

			if result.Result.HasLoginForm != tt.hasLoginForm {
				t.Fatalf("HasLoginForm = %v, want %v", result.Result.HasLoginForm, tt.hasLoginForm)
			}
		})
	}
}

func analyzerBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("ANALYZER_BASE_URL")); value != "" {
		return value
	}

	return "http://web"
}

func fixtureBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("FIXTURE_BASE_URL")); value != "" {
		return value
	}

	return "http://web"
}

func assertHealthOK(t *testing.T, baseURL string) {
	t.Helper()

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("health check failed for %s: %v", baseURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func analyzeFixture(t *testing.T, baseURL string, fixtureURL string) analyzeResponse {
	t.Helper()

	client := &http.Client{Timeout: 10 * time.Second}

	body, err := json.Marshal(analyzeRequest{URL: fixtureURL})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/analyze", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/analyze failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		var response analyzeResponse
		_ = json.NewDecoder(resp.Body).Decode(&response)
		t.Fatalf("status = %d, want %d, error = %#v", resp.StatusCode, http.StatusOK, response.Error)
	}

	var response analyzeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode analyze response: %v", err)
	}

	return response
}
