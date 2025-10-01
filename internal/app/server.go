package app

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Server は http.Server の起動制御をラップする。
type Server struct {
	server *http.Server
}

func NewServer(addr string, handler http.Handler) *Server {
	return &Server{server: &http.Server{Addr: addr, Handler: handler}}
}

// Run はサーバーを起動し、コンテキストキャンセルで優雅停止する。
func (s *Server) Run(ctx context.Context, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return <-errCh
	}
}
