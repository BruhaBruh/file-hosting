package grpcinterceptors

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthorizationInterceptor(apiKey string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Извлекаем метаданные
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Получаем Authorization header
		auth := extractMetadata(md, "authorization")
		expectedAuth := fmt.Sprintf("Bearer %s", apiKey)

		if auth != expectedAuth {
			return nil, status.Error(codes.Unauthenticated, "invalid or missing authorization")
		}

		return handler(ctx, req)
	}
}
