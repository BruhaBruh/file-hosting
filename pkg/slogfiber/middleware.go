package slogfiber

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/trace"
)

type customAttributesCtxKeyType struct{}

var customAttributesCtxKey = customAttributesCtxKeyType{}

var (
	RequestBodyMaxSize  = 64 * 1024 // 64KB
	ResponseBodyMaxSize = 64 * 1024 // 64KB

	// Formatted with http.CanonicalHeaderKey
	RequestIDHeaderKey = "X-Request-Id"
)

type Config struct {
	DefaultLevel     slog.Level
	ClientErrorLevel slog.Level
	ServerErrorLevel slog.Level

	WithUserAgent      bool
	WithRequestID      bool
	WithRequestBody    bool
	WithRequestHeader  bool
	WithResponseBody   bool
	WithResponseHeader bool
	WithSpanID         bool
	WithTraceID        bool

	Filters []Filter
}

// New returns a fiber.Handler (middleware) that logs requests using slog.
//
// Requests with errors are logged using slog.Error().
// Requests without errors are logged using slog.Info().
func New(logger *slog.Logger) fiber.Handler {
	return NewWithConfig(logger, Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,

		WithUserAgent:      false,
		WithRequestID:      true,
		WithRequestBody:    false,
		WithRequestHeader:  false,
		WithResponseBody:   false,
		WithResponseHeader: false,
		WithSpanID:         false,
		WithTraceID:        false,

		Filters: []Filter{},
	})
}

// NewWithFilters returns a fiber.Handler (middleware) that logs requests using slog.
//
// Requests with errors are logged using slog.Error().
// Requests without errors are logged using slog.Info().
func NewWithFilters(logger *slog.Logger, filters ...Filter) fiber.Handler {
	return NewWithConfig(logger, Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,

		WithUserAgent:      false,
		WithRequestID:      true,
		WithRequestBody:    false,
		WithRequestHeader:  false,
		WithResponseBody:   false,
		WithResponseHeader: false,
		WithSpanID:         false,
		WithTraceID:        false,

		Filters: filters,
	})
}

// NewWithConfig returns a fiber.Handler (middleware) that logs requests using slog.
func NewWithConfig(logger *slog.Logger, config Config) fiber.Handler {
	var (
		once       sync.Once
		errHandler fiber.ErrorHandler
	)

	return func(c *fiber.Ctx) error {
		once.Do(func() {
			errHandler = c.App().ErrorHandler
		})

		start := time.Now()

		requestID := c.Get(RequestIDHeaderKey)
		if config.WithRequestID {
			if requestID == "" {
				requestID = uuid.New().String()
			}
			c.Context().SetUserValue("request-id", requestID)
			c.Set("X-Request-ID", requestID)
		}

		ip := c.Context().RemoteIP().String()
		if len(c.IPs()) > 0 {
			ip = c.IPs()[0]
		}

		attrs := attributes{
			start:         start.UTC(),
			method:        string(c.Context().Method()),
			host:          c.Hostname(),
			path:          c.Path(),
			query:         string(c.Request().URI().QueryString()),
			params:        c.AllParams(),
			route:         c.Route().Path,
			referer:       c.Get(fiber.HeaderReferer),
			ip:            ip,
			xForwardedFor: c.IPs(),
			requestID:     requestID,
			attrs:         []slog.Attr{},

			span:   trace.SpanFromContext(c.UserContext()),
			config: &config,
		}

		requestBody := c.Body()
		if len(requestBody) > RequestBodyMaxSize {
			requestBody = requestBody[:RequestBodyMaxSize]
		}
		attrs.request = &requestAttributes{
			length:    len(c.Body()),
			body:      string(requestBody),
			headers:   c.GetReqHeaders(),
			userAgent: string(c.Context().UserAgent()),
		}

		c.SetUserContext(logging.ContextWithLogger(c.UserContext(), logging.WithDefaultAttrs(logger, slog.String("protocol", "http"), attrs.Group())))

		err := c.Next()
		if err != nil {
			if err = errHandler(c, err); err != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError) //nolint:errcheck
			}
		}

		// Pass thru filters and skip early the code below, to prevent unnecessary processing.
		for _, filter := range config.Filters {
			if !filter(c) {
				return err
			}
		}

		responseBody := c.Body()
		if len(responseBody) > ResponseBodyMaxSize {
			responseBody = responseBody[:ResponseBodyMaxSize]
		}

		status := c.Response().StatusCode()

		attrs.response = &responseAttributes{
			status:  status,
			end:     time.Now(),
			length:  len(c.Response().Body()),
			body:    string(responseBody),
			headers: c.GetRespHeaders(),
		}

		// custom context values
		if v := c.Context().UserValue(customAttributesCtxKey); v != nil {
			switch attributes := v.(type) {
			case []slog.Attr:
				attrs.attrs = append(attrs.attrs, attributes...)
			}
		}

		logErr := err
		if logErr == nil {
			logErr = fiber.NewError(status)
		}

		level := config.DefaultLevel
		msg := "Incoming request"
		if status >= http.StatusInternalServerError {
			level = config.ServerErrorLevel
			msg = logErr.Error()
			if msg == "" {
				msg = fmt.Sprintf("HTTP error: %d %s", status, strings.ToLower(http.StatusText(status)))
			}
		} else if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
			level = config.ClientErrorLevel
			msg = logErr.Error()
			if msg == "" {
				msg = fmt.Sprintf("HTTP error: %d %s", status, strings.ToLower(http.StatusText(status)))
			}
		}

		if status != 404 {
			logger.LogAttrs(c.UserContext(), level, msg, slog.String("protocol", "http"), attrs.Group())
		}

		return err
	}
}

// GetRequestID returns the request identifier.
func GetRequestID(c *fiber.Ctx) string {
	return GetRequestIDFromContext(c.Context())
}

// GetRequestIDFromContext returns the request identifier from the context.
func GetRequestIDFromContext(ctx *fasthttp.RequestCtx) string {
	requestID, ok := ctx.UserValue("request-id").(string)
	if !ok {
		return ""
	}

	return requestID
}

// AddCustomAttributes adds custom attributes to the request context.
func AddCustomAttributes(c *fiber.Ctx, attr slog.Attr) {
	v := c.Context().UserValue(customAttributesCtxKey)
	if v == nil {
		c.Context().SetUserValue(customAttributesCtxKey, []slog.Attr{attr})
		return
	}

	switch attrs := v.(type) {
	case []slog.Attr:
		c.Context().SetUserValue(customAttributesCtxKey, append(attrs, attr))
	}
}

func traceID(ctx context.Context) string {
	return trace.SpanFromContext(ctx).SpanContext().TraceID().String()
}

func spanID(ctx context.Context) string {
	return trace.SpanFromContext(ctx).SpanContext().SpanID().String()
}

func extractTraceSpanID(ctx context.Context, withTraceID bool, withSpanID bool) []slog.Attr {
	if !withTraceID && !withSpanID {
		return []slog.Attr{}
	}

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return []slog.Attr{}
	}

	attrs := []slog.Attr{}
	spanCtx := span.SpanContext()

	if withTraceID && spanCtx.HasTraceID() {
		attrs = append(attrs, slog.String(TraceIDKey, traceID(ctx)))
	}

	if withSpanID && spanCtx.HasSpanID() {
		spanID := spanCtx.SpanID().String()
		attrs = append(attrs, slog.String(SpanIDKey, spanID))
	}

	return attrs
}
