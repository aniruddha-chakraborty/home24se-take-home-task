package api

import "net/http"

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {

}
