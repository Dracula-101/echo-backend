package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"shared/pkg/logger"
)

type LoggingInterceptor struct {
	logger logger.Logger
}

func NewLoggingInterceptor(log logger.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{
		logger: log,
	}
}

func (i *LoggingInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := status.Code(err)

		fields := []logger.Field{
			logger.String("method", info.FullMethod),
			logger.String("duration", duration.String()),
			logger.String("code", code.String()),
		}

		if err != nil {
			fields = append(fields, logger.Error(err))
			i.logger.Error("grpc request failed", fields...)
		} else {
			i.logger.Info("grpc request completed", fields...)
		}

		return resp, err
	}
}

func (i *LoggingInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)
		code := status.Code(err)

		fields := []logger.Field{
			logger.String("method", info.FullMethod),
			logger.String("duration", duration.String()),
			logger.String("code", code.String()),
		}

		if err != nil {
			fields = append(fields, logger.Error(err))
			i.logger.Error("grpc stream failed", fields...)
		} else {
			i.logger.Info("grpc stream completed", fields...)
		}

		return err
	}
}
