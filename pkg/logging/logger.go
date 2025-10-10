package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultLevel             = LevelInfo
	defaultAddSource         = true
	defaultIsJSON            = true
	defaultSetDefault        = true
	defaultLogFile           = ""
	defaultLogFileMaxSizeMB  = 10
	defaultLogFileMaxBackups = 3
	defaultLogFileMaxAgeDays = 14
)

func New(opts ...LoggerOption) *Logger {
	config := &LoggerOptions{
		Level:             defaultLevel,
		AddSource:         defaultAddSource,
		IsJSON:            defaultIsJSON,
		SetDefault:        defaultSetDefault,
		LogFilePath:       defaultLogFile,
		LogFileMaxSizeMB:  defaultLogFileMaxSizeMB,
		LogFileMaxBackups: defaultLogFileMaxBackups,
		LogFileMaxAgeDays: defaultLogFileMaxAgeDays,
	}

	for _, opt := range opts {
		opt(config)
	}

	options := &HandlerOptions{
		AddSource: config.AddSource,
		Level:     config.Level,
	}

	// by default we write to stdout.
	var w io.Writer = os.Stdout

	// file or stdout.
	if config.LogFilePath != "" {
		w = &lumberjack.Logger{
			Filename:   config.LogFilePath,
			MaxSize:    config.LogFileMaxSizeMB,
			MaxBackups: config.LogFileMaxBackups,
			MaxAge:     config.LogFileMaxAgeDays,
			Compress:   config.LogFileCompress,
		}
	}

	var h Handler = newTextHandler(w, options)

	if config.IsJSON {
		h = newJSONHandler(w, options)
	}

	logger := new(h)

	if config.SetDefault {
		setDefault(logger)
	}

	return logger
}

// WithAttrs returns logger with attributes.
func WithAttrs(ctx context.Context, attrs ...Attr) *Logger {
	logger := L(ctx)
	return WithDefaultAttrs(logger, attrs...)
}

// WithDefaultAttrs returns logger with default attributes.
func WithDefaultAttrs(logger *Logger, attrs ...Attr) *Logger {
	for _, attr := range attrs {
		logger = logger.With(attr)
	}

	return logger
}

func L(ctx context.Context) *Logger {
	return loggerFromContext(ctx)
}

func Default() *Logger {
	return slog.Default()
}
