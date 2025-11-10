package grpctransport

import (
	"context"
	"fmt"
	"time"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/internal/config"
	"github.com/bruhabruh/file-hosting/internal/domain"
	"github.com/bruhabruh/file-hosting/internal/service"
	"github.com/bruhabruh/file-hosting/pkg/filehosting"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fileHostingServer struct {
	filehosting.UnimplementedFileHostingServer

	config             *config.Config
	logger             *logging.Logger
	fileHostingService service.FileHostingService
}

func newFileHostingServer(config *config.Config, logger *logging.Logger, fileHostingService service.FileHostingService) *fileHostingServer {
	return &fileHostingServer{
		config:             config,
		logger:             logger,
		fileHostingService: fileHostingService,
	}
}

func (s *fileHostingServer) GetFile(ctx context.Context, req *filehosting.FileId) (*filehosting.File, error) {
	file, err := s.fileHostingService.GetFile(ctx, req.GetId())
	if err != nil {
		return nil, apperr.ToGRPCError(err)
	}

	grpcMetadata := make(map[string]*filehosting.MetadataValue)
	for key, values := range file.Metadata.Meta {
		grpcMetadata[key] = &filehosting.MetadataValue{Values: values}
	}

	return &filehosting.File{
		Filename:    file.Metadata.Name,
		Content:     file.Content,
		ContentType: file.Metadata.MimeType,
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

	var backupName *string
	backupName = nil
	if len(metadata.BackupName) > 0 {
		backupName = &metadata.BackupName
	}

	return &filehosting.FileMetadata{
		Name:       metadata.Name,
		MimeType:   metadata.MimeType,
		Sha1:       metadata.Sha1,
		CreatedAt:  metadata.CreatedAt.UTC().Format(time.RFC3339),
		ExpiredAt:  metadata.ExpiredAt.UTC().Format(time.RFC3339),
		BackupName: backupName,
		Meta:       grpcMetadata,
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

	fileName, _, err := s.fileHostingService.UploadFile(ctx, req.GetContent(), metadata, req.GetDuration())
	if err != nil {
		return nil, apperr.ToGRPCError(err)
	}

	return &filehosting.UploadFileResponse{
		Url: fmt.Sprintf("%s/%s", s.config.Origin(), fileName),
		Id:  fileName,
	}, nil
}

func (s *fileHostingServer) GetFiles(ctx context.Context, req *emptypb.Empty) (*filehosting.Files, error) {
	files, err := s.fileHostingService.GetFiles(ctx)
	if err != nil {
		return nil, apperr.ToGRPCError(err)
	}

	metadata := make([]*filehosting.FileMetadata, len(files))
	for i, file := range files {
		meta := make(map[string]*filehosting.MetadataValue)

		for key, values := range file.Meta {
			meta[key] = &filehosting.MetadataValue{Values: values}
		}

		var backupName *string
		backupName = nil
		if len(file.BackupName) > 0 {
			backupName = &file.BackupName
		}

		metadata[i] = &filehosting.FileMetadata{
			Id:         file.Id,
			Name:       file.Name,
			MimeType:   file.MimeType,
			Sha1:       file.Sha1,
			CreatedAt:  file.CreatedAt.UTC().Format(time.RFC3339),
			ExpiredAt:  file.ExpiredAt.UTC().Format(time.RFC3339),
			BackupName: backupName,
			Meta:       meta,
		}
	}

	return &filehosting.Files{
		Metadata: metadata,
	}, nil
}

func (s *fileHostingServer) RenameFile(ctx context.Context, req *filehosting.RenameFileRequest) (*emptypb.Empty, error) {
	if err := s.fileHostingService.RenameFile(ctx, req.GetId(), req.GetNewName()); err != nil {
		return nil, apperr.ToGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *fileHostingServer) DeleteFile(ctx context.Context, req *filehosting.FileId) (*emptypb.Empty, error) {
	if err := s.fileHostingService.DeleteFile(ctx, req.GetId()); err != nil {
		return nil, apperr.ToGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}
