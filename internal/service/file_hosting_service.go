package service

import (
	"context"

	"github.com/bruhabruh/file-hosting/internal/domain"
)

type FileHostingService interface {
	GetFiles(ctx context.Context) ([]*domain.FileMetadata, error)
	GetFile(ctx context.Context, file string) (*domain.File, error)
	GetFileMetadata(ctx context.Context, file string) (*domain.FileMetadata, error)
	UploadFile(ctx context.Context, content []byte, metadata *domain.FileMetadata, rawDuration string) (string, *domain.File, error)
	UploadFileWithGenerativeName(ctx context.Context, content []byte, metadata *domain.FileMetadata, rawDuration string) (string, *domain.File, error)
}
