package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"home24se-take-home/internal/fetcher"
	"home24se-take-home/internal/model"
	"home24se-take-home/internal/parser"
	"home24se-take-home/internal/service"
)

type analyzerService interface {
	Analyze(url string) (model.Result, error)
}

type Handler struct {
	analyzer analyzerService
}

type analyzeRequest struct {
	URL string `json:"url"`
}

type analyzeResponse struct {
	Result *responseResult `json:"result,omitempty"`
	Error  *responseError  `json:"error,omitempty"`
}

type responseResult struct {
	AnalyzedURL   string         `json:"analyzedURL"`
	HTMLVersion   string         `json:"htmlVersion"`
	Title         string         `json:"title"`
	Headings      map[string]int `json:"headings"`
	InternalLinks int            `json:"internalLinks"`
	ExternalLinks int            `json:"externalLinks"`
	BrokenLinks   int            `json:"brokenLinks"`
	HasLoginForm  bool           `json:"hasLoginForm"`
}

type responseError struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

func NewHandler() *Handler {
	return &Handler{
		analyzer: service.New(fetcher.New(), parser.New()),
	}
}

func NewHandlerWithService(analyzer analyzerService) *Handler {
	return &Handler{analyzer: analyzer}
}

func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, analyzeResponse{
			Error: &responseError{
				Code:        http.StatusMethodNotAllowed,
				Description: "method not allowed",
			},
		})
		return
	}

	var req analyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, analyzeResponse{
			Error: &responseError{
				Code:        http.StatusBadRequest,
				Description: "invalid request body",
			},
		})
		return
	}

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, analyzeResponse{
			Error: &responseError{
				Code:        http.StatusBadRequest,
				Description: "url is required",
			},
		})
		return
	}

	result, err := h.analyzer.Analyze(req.URL)
	if err != nil {
		statusCode, description := userFacingError(err)
		writeJSON(w, statusCode, analyzeResponse{
			Error: &responseError{
				Code:        statusCode,
				Description: description,
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, analyzeResponse{
		Result: &responseResult{
			AnalyzedURL:   result.AnalyzedURL,
			HTMLVersion:   result.HTMLVersion,
			Title:         result.Title,
			Headings:      result.Headings,
			InternalLinks: len(result.InternalLinks),
			ExternalLinks: len(result.ExternalLinks),
			BrokenLinks:   result.BrokenLinks,
			HasLoginForm:  result.HasLoginForm,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, response analyzeResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func userFacingError(err error) (int, string) {
	message := err.Error()
	lower := strings.ToLower(message)

	switch {
	case strings.Contains(lower, "please provide a url"):
		return http.StatusBadRequest, "please provide a URL to analyze"
	case strings.Contains(lower, "url format is invalid"):
		return http.StatusBadRequest, "the URL format is invalid"
	case strings.Contains(lower, "no such host"):
		return http.StatusBadRequest, "could not find that website"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline exceeded"):
		return http.StatusGatewayTimeout, "the website took too long to respond"
	default:
		return http.StatusBadRequest, "failed to fetch the webpage"
	}
}
