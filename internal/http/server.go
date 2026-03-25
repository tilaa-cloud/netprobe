package http

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"netprobe/internal/metrics"
)

// Server handles HTTP requests
type Server struct {
	addr      string
	collector *metrics.PrometheusCollector
	server    *http.Server
}

// NewServer creates a new HTTP server
func NewServer(addr string, collector *metrics.PrometheusCollector) *Server {
	return &Server{
		addr:      addr,
		collector: collector,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Register custom collector
	registry := prometheus.NewRegistry()
	if err := registry.Register(s.collector); err != nil {
		return fmt.Errorf("failed to register collector: %w", err)
	}

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", s.handleHealth)

	// Create and start server
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("Starting HTTP server on %s", s.addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// handleHealth handles /health endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
