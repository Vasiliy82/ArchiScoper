package server

import (
	"context"
	"net/http"

	"github.com/Vasiliy82/ArchiScoper/retailer-mockservice/internal/config"

	"github.com/gorilla/mux"
)

type Server struct {
	httpServer *http.Server
}

// New создает новый сервер
func New(cfg config.Config) *Server {
	router := mux.NewRouter()
	router.HandleFunc("/"+cfg.Endpoint.Name, createHandler(cfg.Endpoint)).Methods(cfg.Endpoint.Method)
	router.HandleFunc("/"+cfg.ReverseEndpoint.Name, createHandler(cfg.ReverseEndpoint)).Methods(cfg.Endpoint.Method)

	srv := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: router,
	}

	return &Server{httpServer: srv}
}

// Start запускает HTTP сервер
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown корректно завершает работу сервера
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
