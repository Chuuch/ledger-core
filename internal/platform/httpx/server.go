package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Server struct {
	name string
	http *http.Server
	log  *slog.Logger
}

func NewServer(name, addr string, handler http.Handler, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}

	return &Server{
		name: name,
		log:  log,
		http: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.http.Addr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("http server started", "service", s.name, "addr", ln.Addr().String())
		errCh <- s.http.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		shurdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		s.log.Info("http server shutting down", "service", s.name)
		if err := s.http.Shutdown(shurdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err

	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
