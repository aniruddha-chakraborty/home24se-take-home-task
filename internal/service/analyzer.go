package service

import (
	"fmt"
	"home24se-take-home/internal/fetcher"

	"home24se-take-home/internal/model"
)

type fetcherClient interface {
	Fetch(url string) (*fetcher.Result, error)
}

type parserClient interface {
	Parse(documentURL string, htmlContent string) (model.Result, error)
}

type Analyzer struct {
	fetcher fetcherClient
	parser  parserClient
}

// New creates the analyzer service with the provided dependencies.
func New(fetcher fetcherClient, parser parserClient) *Analyzer {
	return &Analyzer{
		fetcher: fetcher,
		parser:  parser,
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

	return result, nil
}
