package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"shared/pkg/logger"
)

type GrpcServer struct {
	server   *grpc.Server
	listener net.Listener
	logger   logger.Logger
}

type Config struct {
	Port                  int
	MaxConnectionIdle     time.Duration
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration
	Time                  time.Duration
	Timeout               time.Duration
	MaxConcurrentStreams  uint32
	MaxRecvMsgSize        int
	MaxSendMsgSize        int
}

func New(config Config, opts ...grpc.ServerOption) (*GrpcServer, error) {
	defaultOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     config.MaxConnectionIdle,
			MaxConnectionAge:      config.MaxConnectionAge,
			MaxConnectionAgeGrace: config.MaxConnectionAgeGrace,
			Time:                  config.Time,
			Timeout:               config.Timeout,
		}),
		grpc.MaxConcurrentStreams(config.MaxConcurrentStreams),
		grpc.MaxRecvMsgSize(config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(config.MaxSendMsgSize),
	}

	allOpts := append(defaultOpts, opts...)
	server := grpc.NewServer(allOpts...)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &GrpcServer{
		server:   server,
		listener: listener,
	}, nil
}

func (s *GrpcServer) Start() error {
	return s.server.Serve(s.listener)
}

func (s *GrpcServer) Stop(ctx context.Context) error {
	stopped := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		s.server.Stop()
		return ctx.Err()
	case <-stopped:
		return nil
	}
}

func (s *GrpcServer) GetServer() *grpc.Server {
	return s.server
}
