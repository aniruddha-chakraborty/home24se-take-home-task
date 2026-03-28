# Webpage Analyzer

## Overview

This project is a Golang web application for analyzing webpages from a submitted URL.

## Planned Features

- URL input form for submitting a webpage to analyze
- Analyze button to send the request to the server
- HTML version detection
- Page title extraction
- Heading analysis by level (`h1` to `h6`)
- Internal link count
- External link count
- Inaccessible link count
- Login form detection
- Error handling for unreachable URLs
- Error display with HTTP status code and useful description

## Project Structure

```text
cmd/
  server/ <- service starter
    main.go
internal/
  api/ <- HTTP handlers and API-related code
    handler.go
  service/ <- business logic and orchestration
    analyzer.go
  parser/ <- HTML parsing and content extraction logic
    parser.go
  fetcher/ <- remote URL fetching and response access
    fetcher.go
  model/ <- shared request/response and domain models
    result.go
static/ <- frontend assets
  index.html
  app.css
  app.js
```
