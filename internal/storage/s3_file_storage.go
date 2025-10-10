package storage

import (
	"context"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
)

type S3FileStorage struct {
	// fields
}

func NewS3FileStorage() FileStorage {
	return &S3FileStorage{}
}

var _ FileStorage = (*S3FileStorage)(nil)

func (b *S3FileStorage) IsExist(ctx context.Context, file string) bool {
	return false
}

func (b *S3FileStorage) Read(ctx context.Context, file string) ([]byte, error) {
	return nil, apperr.ErrNotImplemented
}

func (b *S3FileStorage) Write(ctx context.Context, file string, data []byte) error {
	return apperr.ErrNotImplemented
}

func (b *S3FileStorage) Move(ctx context.Context, file string, newFile string) error {
	return apperr.ErrNotImplemented
}

func (b *S3FileStorage) Delete(ctx context.Context, file string) error {
	return apperr.ErrNotImplemented
}
