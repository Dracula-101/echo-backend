package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (s *GrpcServer) Run(ctx context.Context, shutdownTimeout time.Duration) error {
	errChan := make(chan error, 1)

	go func() {
		if err := s.Start(); err != nil {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return s.Stop(shutdownCtx)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return s.Stop(shutdownCtx)
	}
}
