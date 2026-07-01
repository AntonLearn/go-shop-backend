// Package server инкапсулирует низкоуровневую настройку сетевого HTTP-сервера,
// изолируя конфигурацию таймаутов и логику запуска от Composition Root.
package server

import (
	"context"
	"net/http"
	"time"
)

const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 10 * time.Second
)

// Server представляет собой обертку над стандартным http.Server.
type Server struct {
	httpServer *http.Server
}

// New создает и настраивает новый экземпляр HTTP-сервера.
func New(port string, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + port,
			Handler:      handler,
			ReadTimeout:  defaultReadTimeout,
			WriteTimeout: defaultWriteTimeout,
		},
	}
}

// Start запускает сервер на прослушивание входящих портов.
// Метод является блокирующим, его следует запускать в отдельной горутине.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown плавно останавливает сервер, позволяя активным запросам завершиться.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
