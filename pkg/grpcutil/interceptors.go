package grpcutil

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryRecoveryInterceptor recovers from panics and returns internal errors.
func UnaryRecoveryInterceptor(log zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("method", info.FullMethod).
					Msg("panic recovered in unary gRPC request")
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// UnaryLoggingInterceptor logs request duration and final status.
func UnaryLoggingInterceptor(log zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)
		evt := log.Info()
		if code != codes.OK {
			evt = log.Warn().Err(err)
		}
		evt.
			Str("method", info.FullMethod).
			Str("code", code.String()).
			Dur("duration", time.Since(start)).
			Msg("gRPC unary request completed")
		return resp, err
	}
}

// UnaryServerOptions returns a standard interceptor chain.
func UnaryServerOptions(log zerolog.Logger) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			UnaryRecoveryInterceptor(log),
			UnaryLoggingInterceptor(log),
		),
	}
}
