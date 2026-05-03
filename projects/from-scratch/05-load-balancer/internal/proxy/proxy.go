// Package proxy implements the reverse proxy using httputil.ReverseProxy.
package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"

	"tour_of_go/projects/from-scratch/05-load-balancer/internal/balancer"
)

// New returns an http.Handler that load-balances across backends using the given strategy.
func New(b balancer.Balancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backend := b.Next()
		if backend == nil {
			http.Error(w, "no healthy backends", http.StatusServiceUnavailable)
			return
		}
		backend.ActiveConns.Add(1)
		defer backend.ActiveConns.Add(-1)

		rp := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = backend.URL.Scheme
				req.URL.Host = backend.URL.Host
				req.Host = backend.URL.Host
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("proxy error to %s: %v", backend.URL.Host, err)
				http.Error(w, "bad gateway", http.StatusBadGateway)
			},
		}
		rp.ServeHTTP(w, r)
	})
}
