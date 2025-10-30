package sloggrpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	RequestIDMetadataKey = "x-request-id"
	TraceIDMetadataKey   = "x-trace-id"
	SpanIDMetadataKey    = "x-span-id"
)

type Config struct {
	DefaultLevel     slog.Level
	ClientErrorLevel slog.Level
	ServerErrorLevel slog.Level

	WithRequestID   bool
	WithTraceID     bool
	WithSpanID      bool
	WithMetadata    bool
	WithRequestBody bool

	Filters []Filter
}

// New returns a grpc.UnaryServerInterceptor that logs requests using slog.
func New(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return NewWithConfig(logger, Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,

		WithRequestID:   true,
		WithTraceID:     false,
		WithSpanID:      false,
		WithMetadata:    false,
		WithRequestBody: false,

		Filters: []Filter{},
	})
}

// NewWithFilters returns a grpc.UnaryServerInterceptor with custom filters.
func NewWithFilters(logger *slog.Logger, filters ...Filter) grpc.UnaryServerInterceptor {
	return NewWithConfig(logger, Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,

		WithRequestID:   true,
		WithTraceID:     false,
		WithSpanID:      false,
		WithMetadata:    false,
		WithRequestBody: false,

		Filters: filters,
	})
}

// NewWithConfig returns a grpc.UnaryServerInterceptor with custom configuration.
func NewWithConfig(logger *slog.Logger, config Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		// Extract metadata from incoming context
		md, _ := metadata.FromIncomingContext(ctx)
		requestID := extractMetadata(md, RequestIDMetadataKey)
		traceID := extractMetadata(md, TraceIDMetadataKey)
		spanID := extractMetadata(md, SpanIDMetadataKey)

		// Generate request ID if not present
		if config.WithRequestID && requestID == "" {
			requestID = uuid.New().String()
		}

		// Build attributes
		attrs := attributes{
			start:      start.UTC(),
			method:     parseMethod(info.FullMethod),
			fullMethod: info.FullMethod,
			requestID:  requestID,
			traceID:    traceID,
			spanID:     spanID,
			metadata:   md,
			attrs:      []slog.Attr{},
			config:     &config,
		}

		if config.WithRequestBody {
			attrs.request = &requestAttributes{
				body: req,
			}
		}

		// Inject logger with default attributes into context
		ctx = logging.ContextWithLogger(ctx, logging.WithDefaultAttrs(logger, attrs.Group()))

		// Call handler
		resp, err := handler(ctx, req)

		// Pass through filters and skip logging if needed
		for _, filter := range config.Filters {
			if !filter(ctx, info) {
				return resp, err
			}
		}

		// Build response attributes
		statusCode := codes.OK
		var statusMsg string

		if err != nil {
			st, _ := status.FromError(err)
			statusCode = st.Code()
			statusMsg = st.Message()
		}

		attrs.response = &responseAttributes{
			status:    statusCode,
			statusMsg: statusMsg,
			end:       time.Now(),
			hasError:  err != nil,
			body:      resp,
		}

		// Determine log level and message
		level := config.DefaultLevel
		msg := "Incoming gRPC request"

		if isServerError(statusCode) {
			level = config.ServerErrorLevel
			msg = statusMsg
			if msg == "" {
				msg = "gRPC server error: " + statusCode.String()
			}
		} else if isClientError(statusCode) {
			level = config.ClientErrorLevel
			msg = statusMsg
			if msg == "" {
				msg = "gRPC client error: " + statusCode.String()
			}
		}

		logger.LogAttrs(ctx, level, msg, slog.String("protocol", "grpc"), attrs.Group())

		return resp, err
	}
}

func extractMetadata(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func parseMethod(fullMethod string) string {
	// fullMethod format: /package.Service/Method
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return fullMethod
}

func isServerError(code codes.Code) bool {
	return code == codes.Internal ||
		code == codes.Unknown ||
		code == codes.DataLoss ||
		code == codes.Unimplemented
}

func isClientError(code codes.Code) bool {
	return code != codes.OK &&
		code != codes.Canceled &&
		!isServerError(code)
}
