package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"shared/pkg/monitoring/metrics"
)

type MetricsInterceptor struct {
	requestCounter    metrics.Counter
	durationHistogram metrics.Histogram
}

func NewMetricsInterceptor(requestCounter metrics.Counter, durationHistogram metrics.Histogram) *MetricsInterceptor {
	return &MetricsInterceptor{
		requestCounter:    requestCounter,
		durationHistogram: durationHistogram,
	}
}

func (i *MetricsInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start).Seconds()
		code := status.Code(err)

		labels := map[string]string{
			"method": info.FullMethod,
			"code":   code.String(),
		}

		i.requestCounter.Inc(labels)
		i.durationHistogram.Observe(duration, labels)

		return resp, err
	}
}

func (i *MetricsInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start).Seconds()
		code := status.Code(err)

		labels := map[string]string{
			"method": info.FullMethod,
			"code":   code.String(),
		}

		i.requestCounter.Inc(labels)
		i.durationHistogram.Observe(duration, labels)

		return err
	}
}
