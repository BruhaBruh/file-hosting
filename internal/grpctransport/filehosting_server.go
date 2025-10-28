package grpctransport

import (
	"context"
	"fmt"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/internal/config"
	"github.com/bruhabruh/file-hosting/internal/domain"
	"github.com/bruhabruh/file-hosting/internal/service"
	"github.com/bruhabruh/file-hosting/pkg/filehosting"
	"github.com/bruhabruh/file-hosting/pkg/logging"
)

type fileHostingServer struct {
	filehosting.UnimplementedFileHostingServer

	config             *config.Config
	logger             *logging.Logger
	fileHostingService *service.FileHostingService
}

func newFileHostingServer(config *config.Config, logger *logging.Logger, fileHostingService *service.FileHostingService) *fileHostingServer {
	return &fileHostingServer{
		config:             config,
		logger:             logger,
		fileHostingService: fileHostingService,
	}
}

func (s *fileHostingServer) GetFile(ctx context.Context, req *filehosting.FileId) (*filehosting.File, error) {
	data, metadata, err := s.fileHostingService.GetFile(ctx, req.GetId())
	if err != nil {
		return nil, apperr.ToGRPCError(err)
	}

	grpcMetadata := make(map[string]*filehosting.MetadataValue)
	for key, values := range metadata.Meta {
		grpcMetadata[key] = &filehosting.MetadataValue{Values: values}
	}

	return &filehosting.File{
		Filename:    metadata.Name,
		Content:     data,
		ContentType: metadata.MimeType,
		Metadata:    grpcMetadata,
	}, nil
}

func (s *fileHostingServer) GetFileMetadata(ctx context.Context, req *filehosting.FileId) (*filehosting.FileMetadata, error) {
	metadata, err := s.fileHostingService.GetFileMetadata(ctx, req.GetId())
	if err != nil {
		return nil, apperr.ToGRPCError(err)
	}

	grpcMetadata := make(map[string]*filehosting.MetadataValue)
	for key, values := range metadata.Meta {
		grpcMetadata[key] = &filehosting.MetadataValue{Values: values}
	}

	return &filehosting.FileMetadata{
		Metadata: grpcMetadata,
	}, nil
}

func (s *fileHostingServer) UploadFile(ctx context.Context, req *filehosting.UploadFileRequest) (*filehosting.UploadFileResponse, error) {
	domainMetadata := make(map[string][]string)
	for key, metadataValue := range req.GetMetadata() {
		data := make([]string, len(metadataValue.GetValues()))
		for i, value := range metadataValue.GetValues() {
			data[i] = value
		}
		domainMetadata[key] = data
	}

	metadata := &domain.FileMetadata{
		Name:     req.GetFilename(),
		MimeType: req.GetContentType(),
		Meta:     domainMetadata,
	}

	fileName, err := s.fileHostingService.UploadFile(ctx, req.GetContent(), metadata, req.GetDuration())
	if err != nil {
		return nil, apperr.ToGRPCError(err)
	}

	return &filehosting.UploadFileResponse{
		Url: fmt.Sprintf("%s/%s", s.config.Origin(), fileName),
		Id:  fileName,
	}, nil
}
