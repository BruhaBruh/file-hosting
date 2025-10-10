package httptransport

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/internal/config"
	"github.com/bruhabruh/file-hosting/internal/service"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/bruhabruh/file-hosting/pkg/slogfiber"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type HttpTransport struct {
	config             *config.Config
	logger             *logging.Logger
	fileHostingService *service.FileHostingService
	fiber              *fiber.App
}

func New(config *config.Config, logger *logging.Logger, fileHostingService *service.FileHostingService) *HttpTransport {
	transport := &HttpTransport{
		config:             config,
		logger:             logger,
		fileHostingService: fileHostingService,
		fiber: fiber.New(
			fiber.Config{
				AppName:               "File-Hosting",
				DisableStartupMessage: true,
				JSONEncoder:           json.Marshal,
				JSONDecoder:           json.Unmarshal,
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
	}

	transport.configureMiddlewares()
	transport.configureRoutes()

	return transport
}

func (ht *HttpTransport) Run() error {
	addr := fmt.Sprintf("0.0.0.0:%d", ht.config.Port())
	ht.logger.Info(
		"Start listening http",
		logging.StringAttr("bound", addr),
	)
	return ht.fiber.Listen(addr)
}

func (ht *HttpTransport) Shutdown() error {
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
					"/livez",
					"/healthz",
				),
			},
		},
	))
	ht.fiber.Use(healthcheck.New())
	ht.fiber.Use(limiter.New(limiter.Config{
		Max:               20,
		Expiration:        30 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))
	ht.fiber.Use(func(c *fiber.Ctx) error {
		logging.ContextWithLogger(c.UserContext(), ht.logger)
		return c.Next()
	})
	ht.fiber.Use(recover.New())
}

func (ht *HttpTransport) configureRoutes() {
	ht.fileRoute()
	ht.fileMetadataRoute()
	ht.uploadPublicRoute()
	ht.uploadPrivateRoute()
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
