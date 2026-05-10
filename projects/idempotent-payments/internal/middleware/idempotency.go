package middleware

import (
	"bytes"
	"net/http"
	"time"

	"tour_of_go/projects/idempotent-payments/internal/domain"
	"tour_of_go/projects/idempotent-payments/internal/ports"
)

// responseRecorder captures the status code, headers, and body written by a handler.
type responseRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// Idempotency returns middleware that deduplicates requests by Idempotency-Key header.
func Idempotency(store ports.IdempotencyStore, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				http.Error(w, "Idempotency-Key header required", http.StatusBadRequest)
				return
			}

			// Cache hit: replay stored response.
			rec, err := store.Get(r.Context(), key)
			if err != nil {
				http.Error(w, "idempotency store error", http.StatusInternalServerError)
				return
			}
			if rec != nil {
				for k, v := range rec.Headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(rec.StatusCode)
				w.Write(rec.Body) //nolint:errcheck
				return
			}

			// Cache miss: capture response, then store it.
			rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rr, r)

			headers := make(map[string]string)
			for k := range w.Header() {
				headers[k] = w.Header().Get(k)
			}
			store.Save(r.Context(), &domain.IdempotencyRecord{ //nolint:errcheck
				Key:        key,
				StatusCode: rr.status,
				Headers:    headers,
				Body:       rr.body.Bytes(),
				ExpiresAt:  time.Now().Add(ttl),
			})
		})
	}
}

// Chain applies middleware left-to-right.
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
