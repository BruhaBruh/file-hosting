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

func (b *BasicFileStorage) IsExist(ctx context.Context, file string) bool {
	_, err := os.Stat(b.path(file))
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (b *BasicFileStorage) Read(ctx context.Context, file string) ([]byte, error) {
	if !b.IsExist(ctx, file) {
		return nil, apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}

	f, err := os.Open(b.path(file))
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

func (b *BasicFileStorage) Write(ctx context.Context, file string, data []byte) error {
	if b.IsExist(ctx, file) {
		return apperr.ErrConflict.WithMessage(fmt.Sprintf("File %s already exist", file))
	}

	f, err := os.Create(b.path(file))
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

func (b *BasicFileStorage) Move(ctx context.Context, file string, newFile string) error {
	if !b.IsExist(ctx, file) {
		return apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}

	err := os.Rename(b.path(file), b.path(newFile))
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail move file %s to %s", file, newFile), logging.ErrAttr(err))
		return apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail move file %s to %s", file, newFile))
	}

	return nil
}

func (b *BasicFileStorage) Delete(ctx context.Context, file string) error {
	if !b.IsExist(ctx, file) {
		return apperr.ErrNotFound.WithMessage(fmt.Sprintf("File %s not found", file))
	}

	err := os.Remove(b.path(file))
	if err != nil {
		logging.L(ctx).Error(fmt.Sprintf("Fail delete file %s", file), logging.ErrAttr(err))
		return apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail delete file %s", file))
	}

	return nil
}

func (b *BasicFileStorage) path(file string) string {
	return path.Join(b.directory, file)
}
