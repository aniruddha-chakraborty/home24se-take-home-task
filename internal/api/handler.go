package api

import "net/http"

func NewHandler() http.Handler {
	return http.NewServeMux()
}
