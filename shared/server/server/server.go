package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shared/pkg/logger"
)

type Server struct {
	httpServer *http.Server
	config     *Config
	logger     logger.Logger
	listener   net.Listener
	tlsConfig  *tls.Config
}

type Config struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
	TLSEnabled      bool
	TLSCertFile     string
	TLSKeyFile      string
	Handler         http.Handler
}

func New(cfg *Config, log logger.Logger) (*Server, error) {
	if cfg.Handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", cfg.Port)
	}

	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 15 * time.Second
	}

	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 15 * time.Second
	}

	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 60 * time.Second
	}

	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 30 * time.Second
	}

	if cfg.MaxHeaderBytes == 0 {
		cfg.MaxHeaderBytes = 1 << 20
	}

	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}

	httpServer := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:        cfg.Handler,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	s := &Server{
		httpServer: httpServer,
		config:     cfg,
		logger:     log,
	}

	if cfg.TLSEnabled {
		if err := s.setupTLS(); err != nil {
			return nil, fmt.Errorf("failed to setup TLS: %w", err)
		}
	}

	return s, nil
}

func (s *Server) setupTLS() error {
	if s.config.TLSCertFile == "" {
		return fmt.Errorf("TLS cert file not provided")
	}

	if s.config.TLSKeyFile == "" {
		return fmt.Errorf("TLS key file not provided")
	}

	cert, err := tls.LoadX509KeyPair(s.config.TLSCertFile, s.config.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	s.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	s.httpServer.TLSConfig = s.tlsConfig

	return nil
}

func (s *Server) Start() error {
	addr := s.httpServer.Addr

	s.logger.Info("Starting HTTP server",
		logger.String("address", addr),
		logger.Bool("tls", s.config.TLSEnabled),
		logger.Duration("read_timeout", s.config.ReadTimeout),
		logger.Duration("write_timeout", s.config.WriteTimeout),
		logger.Duration("idle_timeout", s.config.IdleTimeout),
	)

	if s.config.TLSEnabled {
		return s.httpServer.ListenAndServeTLS("", "")
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) StartWithListener(ln net.Listener) error {
	s.listener = ln

	s.logger.Info("Starting HTTP server with custom listener",
		logger.String("address", ln.Addr().String()),
		logger.Bool("tls", s.config.TLSEnabled),
	)

	if s.config.TLSEnabled {
		return s.httpServer.ServeTLS(ln, "", "")
	}

	return s.httpServer.Serve(ln)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server gracefully")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Graceful shutdown failed", logger.Error(err))
		return err
	}

	s.logger.Info("HTTP server shutdown complete")
	return nil
}

func (s *Server) Close() error {
	s.logger.Warn("Force closing HTTP server")
	return s.httpServer.Close()
}

func (s *Server) ListenAndServe() error {
	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- s.Start()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		s.logger.Info("Shutdown signal received", logger.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to shutdown gracefully, forcing close", logger.Error(err))
			if closeErr := s.Close(); closeErr != nil {
				return fmt.Errorf("failed to force close: %w", closeErr)
			}
			return err
		}

		return nil
	}
}

func (s *Server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.httpServer.Addr
}

func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

func (s *Server) IsRunning() bool {
	return s.listener != nil || s.httpServer != nil
}

type Builder struct {
	config *Config
	logger logger.Logger
}

func NewBuilder() *Builder {
	return &Builder{
		config: &Config{},
	}
}

func (b *Builder) WithHost(host string) *Builder {
	b.config.Host = host
	return b
}

func (b *Builder) WithPort(port int) *Builder {
	b.config.Port = port
	return b
}

func (b *Builder) WithReadTimeout(timeout time.Duration) *Builder {
	b.config.ReadTimeout = timeout
	return b
}

func (b *Builder) WithWriteTimeout(timeout time.Duration) *Builder {
	b.config.WriteTimeout = timeout
	return b
}

func (b *Builder) WithIdleTimeout(timeout time.Duration) *Builder {
	b.config.IdleTimeout = timeout
	return b
}

func (b *Builder) WithShutdownTimeout(timeout time.Duration) *Builder {
	b.config.ShutdownTimeout = timeout
	return b
}

func (b *Builder) WithMaxHeaderBytes(bytes int) *Builder {
	b.config.MaxHeaderBytes = bytes
	return b
}

func (b *Builder) WithTLS(certFile, keyFile string) *Builder {
	b.config.TLSEnabled = true
	b.config.TLSCertFile = certFile
	b.config.TLSKeyFile = keyFile
	return b
}

func (b *Builder) WithHandler(handler http.Handler) *Builder {
	b.config.Handler = handler
	return b
}

func (b *Builder) WithLogger(log logger.Logger) *Builder {
	b.logger = log
	return b
}

func (b *Builder) Build() (*Server, error) {
	if b.logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return New(b.config, b.logger)
}

func ListenAndServe(cfg *Config, log logger.Logger) error {
	srv, err := New(cfg, log)
	if err != nil {
		return err
	}

	return srv.ListenAndServe()
}
