package httptransport

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/internal/config"
	"github.com/bruhabruh/file-hosting/internal/service"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/bruhabruh/file-hosting/pkg/slogfiber"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus"
)

type HttpTransport struct {
	config             *config.Config
	logger             *logging.Logger
	registry           *prometheus.Registry
	fileHostingService service.FileHostingService
	fiber              *fiber.App
	notify             chan error
}

func New(config *config.Config, logger *logging.Logger, registry *prometheus.Registry, fileHostingService service.FileHostingService) *HttpTransport {
	transport := &HttpTransport{
		config:             config,
		registry:           registry,
		logger:             logger,
		fileHostingService: fileHostingService,
		fiber: fiber.New(
			fiber.Config{
				AppName:               "File-Hosting",
				DisableStartupMessage: true,
				JSONEncoder:           json.Marshal,
				JSONDecoder:           json.Unmarshal,
				BodyLimit:             config.HTTP().MaxBodySizeInMB() * 1024 * 1024,
				ErrorHandler: func(c *fiber.Ctx, err error) error {
					code := fiber.StatusInternalServerError

					var e *fiber.Error
					if errors.As(err, &e) {
						code = e.Code
					}
					var apperr *apperr.AppError
					if errors.As(err, &apperr) {
						code = apperr.Code()
					}

					c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

					return c.Status(code).SendString(err.Error())
				},
			},
		),
		notify: make(chan error, 1),
	}

	transport.configureMiddlewares()
	transport.configureRoutes()

	return transport
}

func (ht *HttpTransport) Run() {
	if !ht.config.HTTP().Enabled() {
		return
	}

	addr := fmt.Sprintf("0.0.0.0:%d", ht.config.HTTP().Port())
	ht.logger.Info(
		"Start listening http",
		logging.StringAttr("bound", addr),
	)

	go func() {
		if err := ht.fiber.Listen(addr); err != nil {
			ht.notify <- err
			return
		}
	}()
}

func (ht *HttpTransport) Notify() <-chan error {
	return ht.notify
}

func (ht *HttpTransport) Shutdown() error {
	if !ht.config.GRPC().Enabled() {
		return nil
	}
	return ht.fiber.Shutdown()
}

func (ht *HttpTransport) configureMiddlewares() {
	ht.fiber.Use(slogfiber.NewWithConfig(
		ht.logger,
		slogfiber.Config{
			DefaultLevel:     slog.LevelInfo,
			ClientErrorLevel: slog.LevelWarn,
			ServerErrorLevel: slog.LevelError,

			WithUserAgent:      true,
			WithRequestID:      true,
			WithRequestBody:    false,
			WithRequestHeader:  false,
			WithResponseBody:   false,
			WithResponseHeader: false,
			WithSpanID:         true,
			WithTraceID:        true,

			Filters: []slogfiber.Filter{
				slogfiber.IgnorePath(
					"/.well-known/appspecific/com.chrome.devtools.json",
				),
			},
		},
	))
	ht.fiber.Use(limiter.New(limiter.Config{
		Max:               20,
		Expiration:        30 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))
	ht.fiber.Use(func(c *fiber.Ctx) error {
		logging.ContextWithLogger(c.UserContext(), ht.logger)
		return c.Next()
	})

	fiberProm := fiberprometheus.NewWithRegistry(ht.registry, "file-hosting", "service", "http", make(map[string]string))
	fiberProm.RegisterAt(ht.fiber, "/metrics")
	fiberProm.SetSkipPaths([]string{"/health", "/metrics"})
	fiberProm.SetIgnoreStatusCodes([]int{401, 403, 404})
	ht.fiber.Use(fiberProm.Middleware)

	ht.fiber.Use(recover.New())
}

func (ht *HttpTransport) configureRoutes() {
	ht.healthRoute()
	ht.indexRoute()
	ht.filesRoute()
	ht.fileRoute()
	ht.fileMetadataRoute()
	ht.uploadPublicRoute()
	ht.uploadPrivateRoute()
	ht.renameFileRoute()
	ht.deleteFileRoute()
}

func (ht *HttpTransport) authorizationMiddleware() fiber.Handler {
	header := fmt.Sprintf("Bearer %s", ht.config.ApiKey())

	return func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		if c.Get(fiber.HeaderAuthorization) != header {
			return apperr.ErrUnauthorized
		}

		return c.Next()
	}
}
