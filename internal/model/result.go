package model

type Result struct {
	AnalyzedURL   string
	HTMLVersion   string
	Title         string
	Headings      map[string]int
	InternalLinks []string
	ExternalLinks []string
	BrokenLinks   int
	HasLoginForm  bool
}
