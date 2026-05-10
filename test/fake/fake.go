// Package fake exposes an httptest-backed fake gateway used by the
// SDK's unit tests. It is not part of the public stability contract.
package fake

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// Route registers a handler for (method, path) on a fake gateway.
type Route struct {
	Method  string
	Path    string
	Handler func(w http.ResponseWriter, r *http.Request)
}

// Server bundles an httptest.Server plus a routing table. The zero value
// is not usable; call [NewServer].
type Server struct {
	HTTP *httptest.Server
}

// NewServer constructs a fake gateway with the supplied routes.
func NewServer(routes []Route) *Server {
	mux := http.NewServeMux()
	for _, r := range routes {
		route := r
		mux.HandleFunc(route.Path, func(w http.ResponseWriter, req *http.Request) {
			if route.Method != "" && req.Method != route.Method {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			route.Handler(w, req)
		})
	}
	return &Server{HTTP: httptest.NewServer(mux)}
}

// URL returns the fake server's base URL.
func (s *Server) URL() string { return s.HTTP.URL }

// Close shuts the server down.
func (s *Server) Close() { s.HTTP.Close() }

// WriteEnvelope writes a standard gateway success envelope wrapping data.
func WriteEnvelope(w http.ResponseWriter, requestID string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", requestID)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":    data,
		"status":  200,
		"message": "OK",
	})
}

// WriteError writes a standard gateway error envelope.
func WriteError(w http.ResponseWriter, status int, requestID, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", requestID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":    nil,
		"status":  status,
		"message": code + ": " + message,
	})
}
