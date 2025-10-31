package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/bruhabruh/file-hosting/pkg/s3"
)

type S3FileStorage struct {
	s3 *s3.S3
}

func NewS3FileStorage(s3 *s3.S3) FileStorage {
	return &S3FileStorage{s3: s3}
}

var _ FileStorage = (*S3FileStorage)(nil)

func (s *S3FileStorage) IsExist(ctx context.Context, file string) bool {
	return s.s3.Exists(ctx, file)
}

func (s *S3FileStorage) Files(ctx context.Context) ([]string, error) {
	objects, err := s.s3.Objects(ctx)
	if err != nil {
		return nil, apperr.ErrInternalServerError.WithMessage("Fail get objects in s3")
	}

	files := []string{}

	for _, entry := range objects {
		if strings.HasSuffix(entry, ".metadata") {
			continue
		}
		files = append(files, entry)
	}

	return files, nil
}

func (s *S3FileStorage) Read(ctx context.Context, file string) ([]byte, error) {
	if !s.IsExist(ctx, file) {
		return nil, apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}

	object, err := s.s3.Download(ctx, file)
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail download file %s", file), logging.ErrAttr(err))
		return nil, apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail download file %s", file))
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail read file %s", file), logging.ErrAttr(err))
		return nil, apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail read file %s", file))
	}

	return data, nil
}

func (s *S3FileStorage) Write(ctx context.Context, file string, data []byte, contentType string) error {
	if s.IsExist(ctx, file) {
		return apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s already exist", file))
	}

	reader := bytes.NewReader(data)

	err := s.s3.Upload(ctx, file, reader, reader.Size(), contentType)
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail upload file %s", file), logging.ErrAttr(err))
		return apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail upload to file %s", file))
	}

	return nil
}

func (s *S3FileStorage) Move(ctx context.Context, file string, newFile string) error {
	if !s.IsExist(ctx, file) {
		return apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}

	err := s.s3.Rename(ctx, file, newFile)
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail move file %s to %s", file, newFile), logging.ErrAttr(err))
		return apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail move file %s to %s", file, newFile))
	}

	return nil
}

func (s *S3FileStorage) Delete(ctx context.Context, file string) error {
	if !s.IsExist(ctx, file) {
		return apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}
	return s.s3.Delete(ctx, file)
}
