package service

import (
	"context"
	"time"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/internal/domain"
	"github.com/bruhabruh/file-hosting/internal/storage"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/bruhabruh/file-hosting/pkg/rabbitmq"
	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "file-hosting-service"

var defaultTTL = time.Hour

type FileHostingCachedService struct {
	service FileHostingService
	rdb     *redis.Client
}

func NewFileHostingCachedService(ctx context.Context, fileStorage storage.FileStorage, mq *rabbitmq.RabbitMQ, rdb *redis.Client) (FileHostingService, error) {
	service, err := NewFileHostingService(ctx, fileStorage, mq)
	if err != nil {
		return nil, err
	}
	return &FileHostingCachedService{
		service: service,
		rdb:     rdb,
	}, nil
}

func (s *FileHostingCachedService) GetFiles(ctx context.Context) ([]*domain.FileMetadata, error) {
	val, err := s.rdb.Get(ctx, s.key("files")).Result()
	if err == redis.Nil {
		files, err := s.service.GetFiles(ctx)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(files)
		if err != nil {
			logging.L(ctx).Error("fail marshal files", logging.ErrAttr(err))
		} else {
			if err := s.rdb.Set(ctx, s.key("files"), data, defaultTTL).Err(); err != nil {
				logging.L(ctx).Error("fail cache files", logging.ErrAttr(err))
			}
		}
		return files, nil
	} else if err != nil {
		logging.L(ctx).Error("fail get files", logging.ErrAttr(err))
		return nil, apperr.ErrInternalServerError.WithMessage("fail get files")
	}

	var files []*domain.FileMetadata
	if err := json.Unmarshal([]byte(val), &files); err != nil {
		return nil, apperr.ErrInternalServerError.WithMessage("fail unmarshal files")
	}

	return files, nil
}

func (s *FileHostingCachedService) GetFile(ctx context.Context, filename string) (*domain.File, error) {
	rawFile, err := s.rdb.Get(ctx, s.key("file", filename)).Result()
	if err == redis.Nil {
		file, err := s.service.GetFile(ctx, filename)
		if err != nil {
			return nil, err
		}

		data, err := json.Marshal(file)
		if err != nil {
			logging.L(ctx).Error("fail marshal file", logging.StringAttr("file", filename), logging.ErrAttr(err))
		} else {
			if err := s.rdb.Set(ctx, s.key("file", filename), data, s.ttlOfExpiredAt(file.Metadata.ExpiredAt)).Err(); err != nil {
				logging.L(ctx).Error("fail cache file", logging.StringAttr("file", filename), logging.ErrAttr(err))
			}
		}
		return file, nil
	} else if err != nil {
		logging.L(ctx).Error("fail get file", logging.ErrAttr(err))
		return nil, apperr.ErrInternalServerError.WithMessage("fail get file")
	}

	file, err := domain.NewFileFromBytes([]byte(rawFile))
	if err != nil {
		return nil, apperr.ErrInternalServerError.WithMessage("fail unmarshal file")
	}

	return file, nil
}

func (s *FileHostingCachedService) GetFileMetadata(ctx context.Context, filename string) (*domain.FileMetadata, error) {
	rawFileMetadata, err := s.rdb.Get(ctx, s.key("file", filename, "metadata")).Result()
	if err == redis.Nil {
		fileMetadata, err := s.service.GetFileMetadata(ctx, filename)
		if err != nil {
			return nil, err
		}

		data, err := json.Marshal(fileMetadata)
		if err != nil {
			logging.L(ctx).Error("fail marshal file metadata", logging.StringAttr("file", filename), logging.ErrAttr(err))
		} else {
			if err := s.rdb.Set(ctx, s.key("file", filename, "metadata"), data, s.ttlOfExpiredAt(fileMetadata.ExpiredAt)).Err(); err != nil {
				logging.L(ctx).Error("fail cache file metadata", logging.StringAttr("file", filename), logging.ErrAttr(err))
			}
		}
		return fileMetadata, nil
	} else if err != nil {
		logging.L(ctx).Error("fail get file metadata", logging.ErrAttr(err))
		return nil, apperr.ErrInternalServerError.WithMessage("fail get file metadata")
	}

	fileMetadata, err := domain.NewFileMetadataFromBytes([]byte(rawFileMetadata))
	if err != nil {
		return nil, apperr.ErrInternalServerError.WithMessage("fail unmarshal file metadata")
	}

	return fileMetadata, nil
}

func (s *FileHostingCachedService) UploadFile(ctx context.Context, content []byte, metadata *domain.FileMetadata, rawDuration string) (string, *domain.File, error) {
	filename, file, err := s.service.UploadFile(ctx, content, metadata, rawDuration)
	if err != nil {
		return "", nil, err
	}

	data, err := json.Marshal(file)
	if err != nil {
		logging.L(ctx).Error("fail marshal file", logging.StringAttr("file", filename), logging.ErrAttr(err))
	} else {
		if err := s.rdb.Set(ctx, s.key("file", filename), data, s.ttlOfExpiredAt(file.Metadata.ExpiredAt)).Err(); err != nil {
			logging.L(ctx).Error("fail cache file", logging.StringAttr("file", filename), logging.ErrAttr(err))
		}
	}
	data, err = json.Marshal(file.Metadata)
	if err != nil {
		logging.L(ctx).Error("fail marshal file metadata", logging.StringAttr("file", filename), logging.ErrAttr(err))
	} else {
		if err := s.rdb.Set(ctx, s.key("file", filename, "metadata"), data, s.ttlOfExpiredAt(file.Metadata.ExpiredAt)).Err(); err != nil {
			logging.L(ctx).Error("fail cache file metadata", logging.StringAttr("file", filename), logging.ErrAttr(err))
		}
	}
	if err := s.rdb.Del(ctx, s.key("files")).Err(); err != nil {
		logging.L(ctx).Error("fail delete files cache", logging.ErrAttr(err))
	}

	return filename, file, nil
}

func (s *FileHostingCachedService) UploadFileWithGenerativeName(ctx context.Context, content []byte, metadata *domain.FileMetadata, rawDuration string) (string, *domain.File, error) {
	filename, file, err := s.service.UploadFileWithGenerativeName(ctx, content, metadata, rawDuration)
	if err != nil {
		return "", nil, err
	}

	data, err := json.Marshal(file)
	if err != nil {
		logging.L(ctx).Error("fail marshal file", logging.StringAttr("file", filename), logging.ErrAttr(err))
	} else {
		if err := s.rdb.Set(ctx, s.key("file", filename), data, s.ttlOfExpiredAt(file.Metadata.ExpiredAt)).Err(); err != nil {
			logging.L(ctx).Error("fail cache file", logging.StringAttr("file", filename), logging.ErrAttr(err))
		}
	}
	data, err = json.Marshal(file.Metadata)
	if err != nil {
		logging.L(ctx).Error("fail marshal file metadata", logging.StringAttr("file", filename), logging.ErrAttr(err))
	} else {
		if err := s.rdb.Set(ctx, s.key("file", filename, "metadata"), data, s.ttlOfExpiredAt(file.Metadata.ExpiredAt)).Err(); err != nil {
			logging.L(ctx).Error("fail cache file metadata", logging.StringAttr("file", filename), logging.ErrAttr(err))
		}
	}
	if err := s.rdb.Del(ctx, s.key("files")).Err(); err != nil {
		logging.L(ctx).Error("fail delete files cache", logging.ErrAttr(err))
	}

	return filename, file, nil
}

func (s *FileHostingCachedService) RenameFile(ctx context.Context, oldName string, newName string) error {
	err := s.service.RenameFile(ctx, oldName, newName)
	if err != nil {
		return err
	}
	if err := s.rdb.Del(ctx, s.key("file", oldName)).Err(); err != nil {
		logging.L(ctx).Error("fail delete file", logging.StringAttr("file", oldName), logging.ErrAttr(err))
	}
	if err := s.rdb.Del(ctx, s.key("file", oldName, "metadata")).Err(); err != nil {
		logging.L(ctx).Error("fail delete file metadata", logging.StringAttr("file", oldName), logging.ErrAttr(err))
	}
	if err := s.rdb.Del(ctx, s.key("files")).Err(); err != nil {
		logging.L(ctx).Error("fail delete files cache", logging.ErrAttr(err))
	}
	return nil
}

func (s *FileHostingCachedService) DeleteFile(ctx context.Context, file string) error {
	err := s.service.DeleteFile(ctx, file)
	if err != nil {
		return err
	}
	if err := s.rdb.Del(ctx, s.key("file", file)).Err(); err != nil {
		logging.L(ctx).Error("fail delete file", logging.StringAttr("file", file), logging.ErrAttr(err))
	}
	if err := s.rdb.Del(ctx, s.key("file", file, "metadata")).Err(); err != nil {
		logging.L(ctx).Error("fail delete file metadata", logging.StringAttr("file", file), logging.ErrAttr(err))
	}
	if err := s.rdb.Del(ctx, s.key("files")).Err(); err != nil {
		logging.L(ctx).Error("fail delete files cache", logging.ErrAttr(err))
	}
	return nil
}

func (s *FileHostingCachedService) key(key ...string) string {
	result := redisKeyPrefix
	for _, k := range key {
		result += ":" + k
	}
	return result
}

func (s *FileHostingCachedService) ttl(rawDuration string, allowInfinite ...bool) time.Duration {
	duration := parseDuration(rawDuration, allowInfinite...)
	if duration > defaultTTL {
		duration = defaultTTL
	}
	return duration
}

func (s *FileHostingCachedService) ttlOfExpiredAt(expiredAt time.Time) time.Duration {
	duration := time.Until(expiredAt)
	if duration > defaultTTL {
		duration = defaultTTL
	}
	return duration
}
