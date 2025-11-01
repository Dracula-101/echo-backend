package interceptors

import (
	"context"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"shared/pkg/logger"
)

type RecoveryInterceptor struct {
	logger logger.Logger
}

func NewRecoveryInterceptor(log logger.Logger) *RecoveryInterceptor {
	return &RecoveryInterceptor{
		logger: log,
	}
}

func (i *RecoveryInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				i.logger.Error("panic recovered",
					logger.String("method", info.FullMethod),
					logger.Any("panic", r),
					logger.String("stack", string(stack)),
				)

				err = status.Errorf(codes.Internal, "internal server error: %v", r)
			}
		}()

		return handler(ctx, req)
	}
}

func (i *RecoveryInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				i.logger.Error("panic recovered",
					logger.String("method", info.FullMethod),
					logger.Any("panic", r),
					logger.String("stack", string(stack)),
				)

				err = status.Errorf(codes.Internal, "internal server error: %v", r)
			}
		}()

		return handler(srv, ss)
	}
}
