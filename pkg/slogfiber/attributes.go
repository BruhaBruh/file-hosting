package slogfiber

import (
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

var (
	TraceIDKey   = "trace_id"
	SpanIDKey    = "span_id"
	RequestIDKey = "id"

	HiddenRequestHeaders = map[string]struct{}{
		"authorization": {},
		"cookie":        {},
		"set-cookie":    {},
		"x-auth-token":  {},
		"x-csrf-token":  {},
		"x-xsrf-token":  {},
	}
	HiddenResponseHeaders = map[string]struct{}{
		"set-cookie": {},
	}
)

type attributes struct {
	start         time.Time
	method        string
	host          string
	path          string
	query         string
	params        map[string]string
	route         string
	referer       string
	ip            string
	xForwardedFor []string
	requestID     string

	request  *requestAttributes
	response *responseAttributes
	attrs    []slog.Attr

	span   trace.Span
	config *Config
}

type requestAttributes struct {
	length    int
	body      string
	headers   map[string][]string
	userAgent string
}

type responseAttributes struct {
	status  int
	end     time.Time
	length  int
	body    string
	headers map[string][]string
}

func (a *attributes) Group() slog.Attr {
	attrs := []any{}

	attrs = append(attrs, slog.Time("time", a.start.UTC()))
	attrs = append(attrs, slog.String("method", a.method))
	attrs = append(attrs, slog.String("host", a.host))
	attrs = append(attrs, slog.String("path", a.path))
	attrs = append(attrs, slog.String("query", a.query))
	attrs = append(attrs, slog.Any("params", a.params))
	attrs = append(attrs, slog.String("route", a.route))
	attrs = append(attrs, slog.String("ip", a.ip))
	attrs = append(attrs, slog.Any("x-forwarded-for", a.xForwardedFor))
	attrs = append(attrs, slog.Any("referer", a.referer))
	customAttrs := make([]any, len(a.attrs))
	for i := range a.attrs {
		customAttrs[i] = a.attrs[i]
	}
	attrs = append(attrs, customAttrs...)

	if a.config.WithRequestID {
		attrs = append(attrs, slog.String("request-id", a.requestID))
	}

	if a.request != nil {
		attrs = append(attrs, a.request.group(a.config))
	}
	if a.response != nil {
		attrs = append(attrs, a.response.group(a.config, a.start))
	}

	return slog.Group("http", attrs...)
}

func (a *requestAttributes) group(config *Config) slog.Attr {
	attrs := []any{}

	attrs = append(attrs, slog.Int("length", a.length))

	if config.WithRequestBody {
		attrs = append(attrs, slog.String("body", a.body))
	}

	if config.WithRequestHeader {
		kv := []any{}

		for k, v := range a.headers {
			if _, found := HiddenRequestHeaders[strings.ToLower(k)]; found {
				continue
			}
			kv = append(kv, slog.Any(k, v))
		}

		attrs = append(attrs, slog.Group("header", kv...))
	}

	if config.WithUserAgent {
		attrs = append(attrs, slog.String("user-agent", a.userAgent))
	}

	return slog.Group("request", attrs...)
}

func (a *responseAttributes) group(config *Config, start time.Time) slog.Attr {
	attrs := []any{}

	attrs = append(attrs, slog.Int("length", a.length))
	attrs = append(attrs, slog.Time("time", a.end))
	attrs = append(attrs, slog.Duration("latency", a.end.Sub(start)))
	attrs = append(attrs, slog.Int("status", a.status))

	if config.WithResponseBody {
		attrs = append(attrs, slog.String("body", a.body))
	}

	if config.WithResponseHeader {
		kv := []any{}

		for k, v := range a.headers {
			if _, found := HiddenResponseHeaders[strings.ToLower(k)]; found {
				continue
			}
			kv = append(kv, slog.Any(k, v))
		}

		attrs = append(attrs, slog.Group("header", kv...))
	}

	return slog.Group("response", attrs...)
}
