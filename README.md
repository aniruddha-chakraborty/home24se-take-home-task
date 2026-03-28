# Webpage Analyzer

## Overview

This project is a Golang web application for analyzing webpages from a submitted URL.

## Prerequisites

- Go `1.25` or newer
- Make
- Docker
- Docker Compose

## How To Run

### Install dependencies

```bash
go mod download
```

### Run locally

```bash
make run
```

The application starts on:

```text
http://localhost:8080
```

### Run unit tests

```bash
make unit
```

This runs all Go tests except the integration tests under `test/`.

### Run integration tests

```bash
make integration
```

This will:

- start Docker Compose services
- wait for the services to be ready
- run the integration tests in `test/integration`

## Docker Setup

The integration environment uses two services:

- `analyzer` <- the Go web application
- `web` <- Nginx serving fixture HTML files from `test/html`

You can also start the containers manually with:

```bash
make up
```

and stop them with:

```bash
make down
```

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

## Possible Improvements

- Add a headless-browser fallback for JavaScript-heavy websites, so pages that render most of their content on the client side can still be analyzed more accurately.
- Improve login form detection by supporting a broader multilingual keyword set and stronger form-field heuristics.
- Make blocked or bot-protected websites easier to explain to the user, since some pages may reject non-browser requests even when the URL itself is valid.
- Respect `robots.txt` rules before analyzing a page, so the crawler behavior is more polite and closer to real-world production expectations.
