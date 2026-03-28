package parser

import (
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"

	"home24se-take-home/internal/model"
)

type Parser struct{}

// New creates a parser instance for webpage HTML analysis.
func New() *Parser {
	return &Parser{}
}

// Parse extracts supported analysis fields from the provided HTML document.
func (p *Parser) Parse(documentURL string, htmlContent string) (model.Result, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return model.Result{}, err
	}

	var (
		htmlVersion   string
		title         string
		headings      map[string]int
		hasLogin      bool
		internalLinks []string
		externalLinks []string
		wg            sync.WaitGroup
	)

	wg.Add(5)

	go func() {
		defer wg.Done()
		htmlVersion = detectHTMLVersion(htmlContent)
	}()

	go func() {
		defer wg.Done()
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}()

	go func() {
		defer wg.Done()
		headings = extractHeadings(doc)
	}()

	go func() {
		defer wg.Done()
		hasLogin = hasLoginForm(doc)
	}()

	go func() {
		defer wg.Done()
		internalLinks, externalLinks = extractLinks(doc, documentURL)
	}()

	wg.Wait()

	return model.Result{
		HTMLVersion:   htmlVersion,
		Title:         title,
		Headings:      headings,
		InternalLinks: internalLinks,
		ExternalLinks: externalLinks,
		HasLoginForm:  hasLogin,
	}, nil
}

// detectHTMLVersion inspects the document doctype and maps common values.
func detectHTMLVersion(htmlContent string) string {
	doctypePattern := regexp.MustCompile(`(?is)<!doctype\s+([^>]+)>`)
	match := doctypePattern.FindStringSubmatch(htmlContent)
	if len(match) < 2 {
		return "Unknown"
	}

	doctype := strings.TrimSpace(strings.ToLower(match[1]))

	switch {
	case doctype == "html":
		return "HTML5"
	case strings.Contains(doctype, "html 4.01"):
		return "HTML 4.01"
	case strings.Contains(doctype, "xhtml 1.0"):
		return "XHTML 1.0"
	case strings.Contains(doctype, "xhtml 1.1"):
		return "XHTML 1.1"
	default:
		return strings.TrimSpace(match[1])
	}
}

// extractHeadings counts heading tags from h1 through h6.
func extractHeadings(doc *goquery.Document) map[string]int {
	headings := make(map[string]int, 6)
	for _, level := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		headings[level] = doc.Find(level).Length()
	}
	return headings
}

// extractLinks collects anchor targets and classifies them as internal or external.
func extractLinks(doc *goquery.Document, documentURL string) ([]string, []string) {
	baseURL, err := url.Parse(documentURL)
	if err != nil {
		baseURL = nil
	}

	internalLinks := []string{}
	externalLinks := []string{}

	doc.Find("a[href]").Each(func(_ int, selection *goquery.Selection) {
		href, exists := selection.Attr("href")
		if !exists {
			return
		}

		href = strings.TrimSpace(href)
		if href == "" ||
			strings.HasPrefix(href, "#") ||
			strings.HasPrefix(strings.ToLower(href), "mailto:") ||
			strings.HasPrefix(strings.ToLower(href), "tel:") ||
			strings.HasPrefix(strings.ToLower(href), "javascript:") {
			return
		}

		targetURL, err := url.Parse(href)
		if err != nil {
			return
		}

		switch {
		case strings.HasPrefix(href, "/"):
			internalLinks = append(internalLinks, href)
		case !targetURL.IsAbs():
			internalLinks = append(internalLinks, href)
		case baseURL != nil && strings.EqualFold(targetURL.Hostname(), baseURL.Hostname()):
			internalLinks = append(internalLinks, href)
		default:
			externalLinks = append(externalLinks, href)
		}
	})

	return internalLinks, externalLinks
}

// hasLoginForm checks forms for password inputs or common login-related markers.
func hasLoginForm(doc *goquery.Document) bool {
	if doc.Find(`form input[type="password"]`).Length() > 0 {
		return true
	}

	keywords := []string{"login", "log-in", "signin", "sign-in", "username", "email"}
	found := false

	doc.Find("form").EachWithBreak(func(_ int, form *goquery.Selection) bool {
		formText := strings.ToLower(strings.TrimSpace(form.Text()))
		formAction, _ := form.Attr("action")
		formID, _ := form.Attr("id")
		formClass, _ := form.Attr("class")

		combined := strings.ToLower(strings.Join([]string{formText, formAction, formID, formClass}, " "))
		for _, keyword := range keywords {
			if strings.Contains(combined, keyword) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}
