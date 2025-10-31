package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bruhabruh/file-hosting/internal/config"
	"github.com/bruhabruh/file-hosting/internal/grpctransport"
	"github.com/bruhabruh/file-hosting/internal/httptransport"
	"github.com/bruhabruh/file-hosting/internal/service"
	"github.com/bruhabruh/file-hosting/internal/storage"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/bruhabruh/file-hosting/pkg/rabbitmq"
	"github.com/bruhabruh/file-hosting/pkg/s3"
	"github.com/redis/go-redis/v9"
)

type App struct {
	config *config.Config
}

func New(cfg *config.Config) *App {
	return &App{
		config: cfg,
	}
}

func (a *App) Run() {
	ctx := context.Background()

	logger := a.config.Logger().Build()

	ctx = logging.ContextWithLogger(ctx, logger)

	mq, err := rabbitmq.NewRabbitMQ(a.config.RabbitMQ().URL())
	if err != nil {
		log.Fatalf("Fail create rabbitmq: %s", err.Error())
	}

	s3, err := s3.New(
		a.config.FileStorage().S3().Endpoint(),
		a.config.FileStorage().S3().Region(),
		a.config.FileStorage().S3().AccessKey(),
		a.config.FileStorage().S3().SecretKey(),
		a.config.FileStorage().S3().UseSSL(),
		a.config.FileStorage().S3().Bucket(),
		a.config.FileStorage().S3().Directory(),
	)
	if err != nil {
		log.Fatalf("Fail create s3 client: %s", err.Error())
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     a.config.Redis().URL(),
		Password: a.config.Redis().Password(),
		DB:       a.config.Redis().Database(),
	})

	var fileStorage storage.FileStorage
	if a.config.FileStorage().Basic().Enabled() {
		fileStorage = storage.NewBasicFileStorage(a.config.FileStorage().Basic().Directory())
	}
	if a.config.FileStorage().S3().Enabled() {
		fileStorage = storage.NewS3FileStorage(s3)
	}

	fileHostingService, err := service.NewFileHostingCachedService(ctx, fileStorage, mq, rdb)
	if err != nil {
		log.Fatalf("Fail create file hosting service: %s", err.Error())
	}

	http := httptransport.New(a.config, logger, fileHostingService)
	grpc := grpctransport.New(a.config, logger, fileHostingService)

	http.Run()
	defer func() {
		if err := http.Shutdown(); err != nil {
			logger.Error(err.Error())
		}
	}()
	grpc.Run()
	defer grpc.Shutdown()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		logger.Error(s.String())
	case err = <-http.Notify():
		logger.Error("http server error", logging.ErrAttr(err))
	case err = <-grpc.Notify():
		logger.Error("grpc server error", logging.ErrAttr(err))
	case _ = <-ctx.Done():
		if ctx.Err() != nil {
			logger.Error(ctx.Err().Error())
		}
	}
}
