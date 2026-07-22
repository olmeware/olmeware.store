// Package middleware provides cross-cutting HTTP middleware: request logging,
// CORS, and panic recovery.
package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/httpx"
)

// Chain applies middlewares to h in order, so the first listed runs outermost.
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// statusRecorder captures the response status and byte count for logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Logger prints one line per request to stdout using the standard log package,
// as required: method, path, status, duration, remote IP, and a request id.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := uuid.NewString()[:8]
		w.Header().Set("X-Request-ID", reqID)

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		log.Printf("[%s] %s %s -> %d (%s, %d bytes) ip=%s",
			reqID, r.Method, r.URL.RequestURI(), rec.status,
			time.Since(start).Round(time.Microsecond), rec.bytes, httpx.ClientIP(r))
	})
}

// CORS returns middleware that allows the configured frontend origin and the
// standard localhost dev origins to call the API with credentials.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	allowed := map[string]bool{
		allowedOrigin:           true,
		"http://localhost:3000": true,
		"http://127.0.0.1:3000": true,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowed[strings.TrimRight(origin, "/")] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Guest-Token, Idempotency-Key")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Recover converts panics into a 500 response and logs the failure, so one bad
// request can never take the whole server down.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC recovered: %v (%s %s)", rec, r.Method, r.URL.Path)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"Something went wrong."}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
