package sloggrpc

import (
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

var (
	TraceIDKey   = "trace_id"
	SpanIDKey    = "span_id"
	RequestIDKey = "request_id"

	HiddenMetadataKeys = map[string]struct{}{
		"authorization":               {},
		"x-auth-token":                {},
		"x-api-key":                   {},
		"grpc-metadata-authorization": {},
	}
)

type attributes struct {
	start      time.Time
	method     string
	fullMethod string
	requestID  string
	traceID    string
	spanID     string
	metadata   metadata.MD

	request  *requestAttributes
	response *responseAttributes
	attrs    []slog.Attr

	config *Config
}

type requestAttributes struct {
	body any
}

type responseAttributes struct {
	status    codes.Code
	statusMsg string
	end       time.Time
	hasError  bool
	body      any
}

func (a *attributes) Group() slog.Attr {
	attrs := []any{}

	attrs = append(attrs, slog.Time("time", a.start.UTC()))
	attrs = append(attrs, slog.String("method", a.method))
	attrs = append(attrs, slog.String("full_method", a.fullMethod))

	if a.config.WithRequestID && a.requestID != "" {
		attrs = append(attrs, slog.String(RequestIDKey, a.requestID))
	}

	if a.config.WithTraceID && a.traceID != "" {
		attrs = append(attrs, slog.String(TraceIDKey, a.traceID))
	}

	if a.config.WithSpanID && a.spanID != "" {
		attrs = append(attrs, slog.String(SpanIDKey, a.spanID))
	}

	if a.config.WithMetadata && len(a.metadata) > 0 {
		attrs = append(attrs, a.metadataGroup())
	}

	// Custom attributes
	customAttrs := make([]any, len(a.attrs))
	for i := range a.attrs {
		customAttrs[i] = a.attrs[i]
	}
	attrs = append(attrs, customAttrs...)

	if a.request != nil {
		attrs = append(attrs, a.request.group(a.config))
	}

	if a.response != nil {
		attrs = append(attrs, a.response.group(a.config, a.start))
	}

	return slog.Group("grpc", attrs...)
}

func (a *attributes) metadataGroup() slog.Attr {
	kv := []any{}

	for k, v := range a.metadata {
		if _, found := HiddenMetadataKeys[k]; found {
			continue
		}
		if len(v) == 1 {
			kv = append(kv, slog.String(k, v[0]))
		} else {
			kv = append(kv, slog.Any(k, v))
		}
	}

	return slog.Group("metadata", kv...)
}

func (a *requestAttributes) group(config *Config) slog.Attr {
	attrs := []any{}

	if config.WithRequestBody && a.body != nil {
		attrs = append(attrs, slog.Any("body", a.body))
	}

	return slog.Group("request", attrs...)
}

func (a *responseAttributes) group(config *Config, start time.Time) slog.Attr {
	attrs := []any{}

	attrs = append(attrs, slog.Time("time", a.end))
	attrs = append(attrs, slog.Duration("latency", a.end.Sub(start)))
	attrs = append(attrs, slog.String("status", a.status.String()))

	if a.hasError && a.statusMsg != "" {
		attrs = append(attrs, slog.String("error", a.statusMsg))
	}

	if config.WithRequestBody && a.body != nil {
		attrs = append(attrs, slog.Any("body", a.body))
	}

	return slog.Group("response", attrs...)
}
