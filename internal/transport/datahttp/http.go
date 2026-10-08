// Package datahttp implements the private authenticated memory API.
package datahttp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/mcp-runtime/cully/internal/memory"
)

type Envelope struct {
	Owner   string         `json:"owner"`
	Request memory.Request `json:"request"`
}

func Handler(service memory.Service, token string, ready func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready != nil && ready(r.Context()) != nil {
			write(w, 503, map[string]string{"error": "not ready"})
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("POST /v1/memory", func(w http.ResponseWriter, r *http.Request) {
		fields := strings.Fields(r.Header.Get("Authorization"))
		if token == "" || len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") || subtle.ConstantTimeCompare([]byte(fields[1]), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			write(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var in Envelope
		if err := d.Decode(&in); err != nil {
			write(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		var extra any
		if d.Decode(&extra) != io.EOF {
			write(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		result, err := service.Execute(r.Context(), in.Owner, in.Request)
		if err != nil {
			status := 503
			if errors.Is(err, memory.ErrInvalid) {
				status = 400
			} else if errors.Is(err, memory.ErrForbidden) {
				status = 403
			} else if errors.Is(err, memory.ErrConflict) {
				status = 409
			}
			write(w, status, map[string]string{"error": http.StatusText(status)})
			return
		}
		write(w, 200, result)
	})
	return mux
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
