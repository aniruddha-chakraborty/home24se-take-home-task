package model

type Result struct {
	HTMLVersion   string
	Title         string
	Headings      map[string]int
	InternalLinks []string
	ExternalLinks []string
	HasLoginForm  bool
}
