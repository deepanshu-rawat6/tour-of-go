package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"tour_of_go/projects/from-scratch/11-api-gateway/internal/middleware"
)

// New returns a ReverseProxy that forwards to target.
// It injects X-Internal-User-Id and X-Request-ID from context,
// and strips the Authorization header so downstream services never see the JWT.
func New(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		panic("proxy: invalid target URL: " + target)
	}
	rp := httputil.NewSingleHostReverseProxy(u)
	orig := rp.Director
	rp.Director = func(r *http.Request) {
		orig(r)
		// Strip JWT — downstream services trust X-Internal-User-Id instead.
		r.Header.Del("Authorization")

		// Inject identity from context (set by Auth middleware).
		if claims, ok := middleware.ClaimsFromContext(r.Context()); ok {
			r.Header.Set("X-Internal-User-Id", claims.UserID)
			r.Header.Set("X-Internal-User-Role", claims.Role)
		}

		// Propagate trace ID.
		if id := middleware.RequestIDFromContext(r.Context()); id != "" {
			r.Header.Set("X-Request-ID", id)
		}
	}
	return rp
}
