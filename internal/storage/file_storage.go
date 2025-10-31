package storage

import "context"

type FileStorage interface {
	IsExist(ctx context.Context, file string) bool
	Files(ctx context.Context) ([]string, error)
	Read(ctx context.Context, file string) ([]byte, error)
	Write(ctx context.Context, file string, data []byte, contentType string) error
	Move(ctx context.Context, file string, newFile string) error
	Delete(ctx context.Context, file string) error
}
