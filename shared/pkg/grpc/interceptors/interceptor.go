package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

type UnaryServerInterceptor = grpc.UnaryServerInterceptor
type StreamServerInterceptor = grpc.StreamServerInterceptor
type UnaryClientInterceptor = grpc.UnaryClientInterceptor
type StreamClientInterceptor = grpc.StreamClientInterceptor

func ChainUnaryServer(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chain
			chain = func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return interceptor(currentCtx, currentReq, info, next)
			}
		}
		return chain(ctx, req)
	}
}

func ChainUnaryClient(interceptors ...grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		chain := invoker
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chain
			chain = func(currentCtx context.Context, currentMethod string, currentReq, currentReply interface{}, currentConn *grpc.ClientConn, currentOpts ...grpc.CallOption) error {
				return interceptor(currentCtx, currentMethod, currentReq, currentReply, currentConn, next, currentOpts...)
			}
		}
		return chain(ctx, method, req, reply, cc, opts...)
	}
}
