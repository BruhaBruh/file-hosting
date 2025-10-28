package grpcinterceptors

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	requestIDKey = "x-request-id"
	traceIDKey   = "x-trace-id"
	spanIDKey    = "x-span-id"
)

func LoggerInterceptor(
	baseLogger *slog.Logger,
	injector func(context.Context, *slog.Logger) context.Context,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		// Извлекаем метаданные из запроса
		md, _ := metadata.FromIncomingContext(ctx)
		requestID := extractMetadata(md, requestIDKey)
		traceID := extractMetadata(md, traceIDKey)
		spanID := extractMetadata(md, spanIDKey)

		if len(requestID) > 0 {
			uuid, err := uuid.NewRandom()
			if err == nil {
				requestID = uuid.String()
			}
		}

		// Парсим метод (например: /filehosting.FileHosting/UploadFile -> UploadFile)
		method := parseMethod(info.FullMethod)

		// Создаём логгер с базовыми атрибутами
		requestLogger := baseLogger.With(
			slog.String("method", method),
			slog.String("full_method", info.FullMethod),
		)
		if len(requestID) > 0 {
			requestLogger = requestLogger.With(slog.String("request_id", requestID))
		}
		if len(traceID) > 0 {
			requestLogger = requestLogger.With(slog.String("trace_id", traceID))
		}
		if len(spanID) > 0 {
			requestLogger = requestLogger.With(slog.String("span_id", spanID))
		}

		// Добавляем логгер в контекст через injector
		ctx = injector(ctx, requestLogger)

		// Вызываем обработчик
		resp, err := handler(ctx, req)

		// Логируем результат
		duration := time.Since(start)
		level := slog.LevelInfo
		statusCode := codes.OK
		statusMsg := "OK"

		if err != nil {
			st, _ := status.FromError(err)
			statusCode = st.Code()
			statusMsg = st.Message()

			// Определяем уровень логирования по коду ошибки
			switch statusCode {
			case codes.Internal, codes.Unknown, codes.DataLoss:
				level = slog.LevelError
			default:
				level = slog.LevelWarn
			}

			requestLogger.LogAttrs(ctx, level,
				"grpc request failed",
				slog.String("code", statusCode.String()),
				slog.String("message", statusMsg),
				slog.Duration("duration", duration),
			)
		} else {
			requestLogger.LogAttrs(ctx, level,
				"grpc request completed",
				slog.String("code", statusCode.String()),
				slog.Duration("duration", duration),
			)
		}

		return resp, err
	}
}

func parseMethod(fullMethod string) string {
	parts := strings.Split(fullMethod, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullMethod
}
