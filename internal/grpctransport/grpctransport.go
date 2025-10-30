package grpctransport

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/bruhabruh/file-hosting/internal/config"
	"github.com/bruhabruh/file-hosting/internal/service"
	"github.com/bruhabruh/file-hosting/pkg/filehosting"
	"github.com/bruhabruh/file-hosting/pkg/grpcinterceptors"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/bruhabruh/file-hosting/pkg/sloggrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCTransport struct {
	config             *config.Config
	logger             *logging.Logger
	fileHostingService *service.FileHostingService
	grpc               *grpc.Server
	notify             chan error
}

func New(config *config.Config, logger *logging.Logger, fileHostingService *service.FileHostingService) *GRPCTransport {
	transport := &GRPCTransport{
		config:             config,
		logger:             logger,
		fileHostingService: fileHostingService,
		grpc: grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				sloggrpc.NewWithConfig(
					logger,
					sloggrpc.Config{
						DefaultLevel:     slog.LevelInfo,
						ClientErrorLevel: slog.LevelWarn,
						ServerErrorLevel: slog.LevelError,
						WithRequestID:    true,
						WithSpanID:       true,
						WithTraceID:      true,
						WithMetadata:     false,
						WithRequestBody:  false,
						Filters:          []sloggrpc.Filter{},
					}),
				grpcinterceptors.UnaryServerAuthorizationInterceptor(config.ApiKey()),
			),
		),
		notify: make(chan error, 1),
	}

	filehosting.RegisterFileHostingServer(transport.grpc, newFileHostingServer(config, logger, fileHostingService))

	reflection.Register(transport.grpc)

	return transport
}

func (gt *GRPCTransport) Run() {
	if !gt.config.GRPC().Enabled() {
		return
	}

	addr := fmt.Sprintf("0.0.0.0:%d", gt.config.GRPC().Port())
	gt.logger.Info(
		"Start listening grpc",
		logging.StringAttr("bound", addr),
	)

	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", gt.config.GRPC().Port()))
		if err != nil {
			gt.notify <- err
			return
		}

		err = gt.grpc.Serve(listener)
		if err != nil {
			gt.notify <- err
			return
		}
	}()
}

func (ht *GRPCTransport) Notify() <-chan error {
	return ht.notify
}

func (gt *GRPCTransport) Shutdown() {
	if !gt.config.GRPC().Enabled() {
		return
	}
	gt.grpc.GracefulStop()
}
