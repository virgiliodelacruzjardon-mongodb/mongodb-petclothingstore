package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"petstore/internal/db"
)

type Handler struct {
	S *db.Store
}

func New(s *db.Store) *Handler { return &Handler{S: s} }

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// paginate reads ?page= and ?limit= query params with sane defaults.
func paginate(r *http.Request) (page, limit int64) {
	page = 1
	limit = 20
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.ParseInt(p, 10, 64); err == nil && n > 0 {
			page = n
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.ParseInt(l, 10, 64); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	return
}

type listResponse struct {
	Data  interface{} `json:"data"`
	Page  int64       `json:"page"`
	Limit int64       `json:"limit"`
	Total int64       `json:"total"`
}
