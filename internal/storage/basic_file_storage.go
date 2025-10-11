package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/pkg/logging"
)

type BasicFileStorage struct {
	directory string
}

func NewBasicFileStorage(directory string) FileStorage {
	if err := os.MkdirAll(directory, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	return &BasicFileStorage{
		directory: directory,
	}
}

var _ FileStorage = (*BasicFileStorage)(nil)

func (s *BasicFileStorage) IsExist(ctx context.Context, file string) bool {
	_, err := os.Stat(s.path(file))
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (s *BasicFileStorage) Read(ctx context.Context, file string) ([]byte, error) {
	if !s.IsExist(ctx, file) {
		return nil, apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}

	f, err := os.Open(s.path(file))
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail open file %s", file), logging.ErrAttr(err))
		return nil, apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail open file %s", file))
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail read file %s", file), logging.ErrAttr(err))
		return nil, apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail read file %s", file))
	}

	return data, nil
}

func (s *BasicFileStorage) Write(ctx context.Context, file string, data []byte, contentType string) error {
	if s.IsExist(ctx, file) {
		return apperr.ErrConflict.WithMessage(fmt.Sprintf("File %s already exist", file))
	}

	f, err := os.Create(s.path(file))
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail create file %s", file), logging.ErrAttr(err))
		return apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail create file %s", file))
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail write file %s", file), logging.ErrAttr(err))
		return apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail write to file %s", file))
	}

	return nil
}

func (s *BasicFileStorage) Move(ctx context.Context, file string, newFile string) error {
	if !s.IsExist(ctx, file) {
		return apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}

	err := os.Rename(s.path(file), s.path(newFile))
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail move file %s to %s", file, newFile), logging.ErrAttr(err))
		return apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail move file %s to %s", file, newFile))
	}

	return nil
}

func (s *BasicFileStorage) Delete(ctx context.Context, file string) error {
	if !s.IsExist(ctx, file) {
		return apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}

	err := os.Remove(s.path(file))
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail delete file %s", file), logging.ErrAttr(err))
		return apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail delete file %s", file))
	}

	return nil
}

func (s *BasicFileStorage) path(file string) string {
	return path.Join(s.directory, file)
}
