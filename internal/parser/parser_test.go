package parser

import (
	"testing"
)

/*
TestParse_HTML5Document validates the main happy-path behavior using a
single HTML5 document that contains a title, multiple heading levels,
internal links, an external link, and a password field inside a form.
It verifies that Parse() fills all major result fields correctly in one run.
*/
func TestParse_HTML5Document(t *testing.T) {
	t.Parallel()

	p := New()

	html := `
<!DOCTYPE html>
<html>
  <head>
    <title>Example Title</title>
  </head>
  <body>
    <h1>Main</h1>
    <h2>Section A</h2>
    <h2>Section B</h2>
    <h6>Fine Print</h6>

    <a href="/internal">Internal Root</a>
    <a href="relative/path">Internal Relative</a>
    <a href="https://example.com/about">Internal Absolute</a>
    <a href="https://golang.org">External</a>

    <form>
      <input type="password" />
    </form>
  </body>
</html>`

	result, err := p.Parse("https://example.com", html)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.HTMLVersion != "HTML5" {
		t.Fatalf("HTMLVersion = %q, want %q", result.HTMLVersion, "HTML5")
	}

	if result.Title != "Example Title" {
		t.Fatalf("Title = %q, want %q", result.Title, "Example Title")
	}

	assertHeadingCount(t, result.Headings, "h1", 1)
	assertHeadingCount(t, result.Headings, "h2", 2)
	assertHeadingCount(t, result.Headings, "h3", 0)
	assertHeadingCount(t, result.Headings, "h6", 1)

	if got := len(result.InternalLinks); got != 3 {
		t.Fatalf("len(InternalLinks) = %d, want 3; got %#v", got, result.InternalLinks)
	}

	if got := len(result.ExternalLinks); got != 1 {
		t.Fatalf("len(ExternalLinks) = %d, want 1; got %#v", got, result.ExternalLinks)
	}

	if !result.HasLoginForm {
		t.Fatal("HasLoginForm = false, want true")
	}
}

/*
TestParse_LoginFormDetectedWithoutPasswordField checks the secondary login
detection path. Instead of relying on a password input, it uses form action,
class, and text content containing login-related keywords to confirm that the
parser still marks the page as containing a login form.
*/
func TestParse_LoginFormDetectedWithoutPasswordField(t *testing.T) {
	t.Parallel()

	p := New()

	html := `
<!DOCTYPE html>
<html>
  <body>
    <form action="/signin" class="user-login">
      <label>Email</label>
      <input type="text" name="email" />
    </form>
  </body>
</html>`

	result, err := p.Parse("https://example.com", html)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !result.HasLoginForm {
		t.Fatal("HasLoginForm = false, want true")
	}
}

/*
TestParse_NoDoctypeNoTitleNoLogin verifies the fallback behavior for sparse
documents. It confirms that the parser returns "Unknown" for HTML version,
an empty title when no title exists, correct heading counts, no collected
links for ignored href schemes, and no false positive login detection.
*/
func TestParse_NoDoctypeNoTitleNoLogin(t *testing.T) {
	t.Parallel()

	p := New()

	html := `
		<html>
		  <body>
			<h3>Only heading</h3>
			<a href="#section">Fragment</a>
			<a href="mailto:test@example.com">Mail</a>
			<a href="tel:+4912345">Phone</a>
			<a href="javascript:void(0)">JS</a>
		  </body>
		</html>`

	result, err := p.Parse("https://example.com", html)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.HTMLVersion != "Unknown" {
		t.Fatalf("HTMLVersion = %q, want %q", result.HTMLVersion, "Unknown")
	}

	if result.Title != "" {
		t.Fatalf("Title = %q, want empty", result.Title)
	}

	assertHeadingCount(t, result.Headings, "h3", 1)
	assertHeadingCount(t, result.Headings, "h1", 0)

	if len(result.InternalLinks) != 0 {
		t.Fatalf("InternalLinks = %#v, want none", result.InternalLinks)
	}

	if len(result.ExternalLinks) != 0 {
		t.Fatalf("ExternalLinks = %#v, want none", result.ExternalLinks)
	}

	if result.HasLoginForm {
		t.Fatal("HasLoginForm = true, want false")
	}
}

/*
TestParse_DoctypeVariants uses a table-driven approach to cover several
common and uncommon doctype declarations. This confirms that the doctype
mapping logic handles known versions and preserves unknown values.
*/
func TestParse_DoctypeVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "html 4.01",
			html: `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN">`,
			want: "HTML 4.01",
		},
		{
			name: "xhtml 1.0",
			html: `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN">`,
			want: "XHTML 1.0",
		},
		{
			name: "xhtml 1.1",
			html: `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN">`,
			want: "XHTML 1.1",
		},
		{
			name: "unknown doctype preserved",
			html: `<!DOCTYPE custom-system>`,
			want: "custom-system",
		},
	}

	p := New()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := p.Parse("https://example.com", tt.html)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if result.HTMLVersion != tt.want {
				t.Fatalf("HTMLVersion = %q, want %q", result.HTMLVersion, tt.want)
			}
		})
	}
}

/*
TestParse_InvalidBaseURLStillClassifiesLinks ensures the parser behaves
safely even when the document URL is invalid. In that case, relative and
root-based links should still be treated as internal, while absolute links
to another host remain external.
*/
func TestParse_InvalidBaseURLStillClassifiesLinks(t *testing.T) {
	t.Parallel()

	p := New()

	html := `
<!DOCTYPE html>
<html>
  <body>
    <a href="/root-link">Internal root</a>
    <a href="relative-link">Internal relative</a>
    <a href="https://golang.org">External</a>
  </body>
</html>`

	result, err := p.Parse("://bad-url", html)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := len(result.InternalLinks); got != 2 {
		t.Fatalf("len(InternalLinks) = %d, want 2; got %#v", got, result.InternalLinks)
	}

	if got := len(result.ExternalLinks); got != 1 {
		t.Fatalf("len(ExternalLinks) = %d, want 1; got %#v", got, result.ExternalLinks)
	}
}

/*
TestParse_MalformedHTMLStillParses verifies resilience against imperfect
markup. goquery can often recover from broken HTML, so this test ensures the
parser still extracts useful data instead of failing on minor document issues.
*/
func TestParse_MalformedHTMLStillParses(t *testing.T) {
	t.Parallel()

	p := New()

	html := `<!DOCTYPE html>
<html>
  <head>
    <title>Broken</title>
  </head>
  <body>
    <h1>One
    <a href="/x">X</a>
  </body>
</html>`

	result, err := p.Parse("https://example.com", html)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.HTMLVersion != "HTML5" {
		t.Fatalf("HTMLVersion = %q, want %q", result.HTMLVersion, "HTML5")
	}

	assertHeadingCount(t, result.Headings, "h1", 1)

	if got := len(result.InternalLinks); got != 1 {
		t.Fatalf("len(InternalLinks) = %d, want 1; got %#v", got, result.InternalLinks)
	}
}

/*
assertHeadingCount is a small test helper that keeps heading-count checks
readable and gives clearer failure messages when a specific heading level is
missing or has an unexpected value.
*/
func assertHeadingCount(t *testing.T, headings map[string]int, level string, want int) {
	t.Helper()

	got, ok := headings[level]
	if !ok {
		t.Fatalf("missing heading level %q", level)
	}

	if got != want {
		t.Fatalf("Headings[%q] = %d, want %d", level, got, want)
	}
}
