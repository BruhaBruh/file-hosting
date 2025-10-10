package app

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bruhabruh/file-hosting/internal/config"
	"github.com/bruhabruh/file-hosting/internal/httptransport"
	"github.com/bruhabruh/file-hosting/internal/service"
	"github.com/bruhabruh/file-hosting/internal/storage"
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
	logger := a.config.Logger().Build()

	var fileStorage storage.FileStorage
	if a.config.FileStorage().Basic().Enabled() {
		fileStorage = storage.NewBasicFileStorage(a.config.FileStorage().Basic().Directory())
	}

	fileHostingService := service.NewFileHostingService(fileStorage)

	http := httptransport.New(a.config, logger, fileHostingService)

	if err := http.Run(); err != nil {
		log.Fatalf("Fail run http transport: %s", err.Error())
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	s := <-interrupt
	logger.Error(s.String())

	if err := http.Shutdown(); err != nil {
		logger.Error(err.Error())
	}
}
