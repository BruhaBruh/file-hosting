package service

import (
	"context"
	"crypto/sha1"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/internal/domain"
	"github.com/bruhabruh/file-hosting/internal/storage"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/bruhabruh/file-hosting/pkg/rabbitmq"
	"github.com/goccy/go-json"
	"github.com/streadway/amqp"
)

const (
	fileDeletionQueueName = "file-hosting-service/delete-file"
	defaultFileDuration   = time.Hour
)

var infiniteTimeStamp = time.Unix(0, 0)

type deleteFileMessage struct {
	FileName  string    `json:"fileName"`
	Sha1      string    `json:"sha1"`
	ExpiredAt time.Time `json:"expiredAt"`
}

type FileHostingService struct {
	ctx         context.Context
	fileStorage storage.FileStorage
	mq          *rabbitmq.RabbitMQ
}

func NewFileHostingService(ctx context.Context, fileStorage storage.FileStorage, mq *rabbitmq.RabbitMQ) (*FileHostingService, error) {
	service := &FileHostingService{
		ctx:         ctx,
		fileStorage: fileStorage,
		mq:          mq,
	}

	err := service.mq.DeclareQueue(fileDeletionQueueName)
	if err != nil {
		return nil, err
	}

	service.mq.Consume(service.ctx, fileDeletionQueueName, service.handleDeleteFileMessage)

	return service, nil
}

func (s *FileHostingService) GetFile(ctx context.Context, file string) ([]byte, *domain.FileMetadata, error) {
	data, err := s.fileStorage.Read(ctx, file)
	if err != nil {
		return nil, nil, err
	}

	metadata, err := s.GetFileMetadata(ctx, file)
	if err != nil {
		return nil, nil, err
	}

	return data, metadata, nil
}

func (s *FileHostingService) GetFileMetadata(ctx context.Context, file string) (*domain.FileMetadata, error) {
	data, err := s.fileStorage.Read(ctx, s.metadataFile(file))
	if err != nil {
		return nil, err
	}
	metadata, err := domain.NewFileMetadataFromBytes(data)
	if err != nil {
		return nil, apperr.ErrInternalServerError.WithMessage(fmt.Sprintf("Fail read metadata of file %s", file))
	}
	return metadata, nil
}

func (s *FileHostingService) UploadFile(ctx context.Context, data []byte, metadata *domain.FileMetadata, rawDuration string) (string, error) {
	now := time.Now()

	expiredAt := infiniteTimeStamp
	if duration := s.parseDuration(rawDuration, true); duration != 0 {
		expiredAt = now.Add(duration)
	}

	if s.fileStorage.IsExist(ctx, metadata.Name) {
		oldFileData, err := s.fileStorage.Read(ctx, metadata.Name)
		if err == nil {
			newSha1 := s.sha1(data)
			oldSha1 := s.sha1(oldFileData)
			if newSha1 == oldSha1 {
				return metadata.Name, nil
			}
		}
		newFileName := fmt.Sprintf("%s.%d", metadata.Name, now.UnixNano())
		newMetadataFileName := fmt.Sprintf("%s.%d.metadata", metadata.Name, now.UnixNano())

		oldMetadata, _ := s.GetFileMetadata(ctx, metadata.Name)
		if oldMetadata != nil && oldMetadata.ExpiredAt != infiniteTimeStamp {
			err = s.scheduleDeleteFile(newFileName, oldMetadata.Sha1, oldMetadata.ExpiredAt)
			if err != nil {
				return "", err
			}
		}

		if err := s.fileStorage.Move(ctx, metadata.Name, newFileName); err != nil {
			return "", err
		}
		if err := s.fileStorage.Move(ctx, s.metadataFile(metadata.Name), newMetadataFileName); err != nil {
			s.fileStorage.Move(ctx, newFileName, metadata.Name)
			return "", err
		}
		metadata.BackupName = newFileName
	}

	newMetadata := &domain.FileMetadata{
		Name:       metadata.Name,
		MimeType:   metadata.MimeType,
		Sha1:       s.sha1(data),
		Meta:       metadata.Meta,
		CreatedAt:  now,
		ExpiredAt:  expiredAt,
		BackupName: metadata.BackupName,
	}

	metadataInBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return "", apperr.ErrInternalServerError.WithMessage("Fail serialize metadata")
	}

	if newMetadata.ExpiredAt != infiniteTimeStamp {
		err = s.scheduleDeleteFile(newMetadata.Name, newMetadata.Sha1, newMetadata.ExpiredAt)
		if err != nil {
			return "", err
		}
	}

	err = s.fileStorage.Write(ctx, newMetadata.Name, data)
	if err != nil {
		return "", err
	}

	err = s.fileStorage.Write(ctx, s.metadataFile(newMetadata.Name), metadataInBytes)
	if err != nil {
		s.fileStorage.Delete(ctx, newMetadata.Name)
		return "", err
	}

	return newMetadata.Name, nil
}

func (s *FileHostingService) UploadFileWithGenerativeName(ctx context.Context, data []byte, metadata *domain.FileMetadata, rawDuration string) (string, error) {
	fileName := s.generateFileName()
	for {
		if s.fileStorage.IsExist(ctx, fileName) {
			fileName = s.generateFileName()
			continue
		}
		break
	}

	now := time.Now()
	expiredAt := now.Add(s.parseDuration(rawDuration))

	newMetadata := &domain.FileMetadata{
		Name:       metadata.Name,
		MimeType:   metadata.MimeType,
		Sha1:       s.sha1(data),
		Meta:       metadata.Meta,
		CreatedAt:  now,
		ExpiredAt:  expiredAt,
		BackupName: metadata.BackupName,
	}

	metadataInBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return "", apperr.ErrInternalServerError.WithMessage("Fail serialize metadata")
	}

	err = s.scheduleDeleteFile(fileName, newMetadata.Sha1, newMetadata.ExpiredAt)
	if err != nil {
		return "", err
	}

	err = s.fileStorage.Write(ctx, fileName, data)
	if err != nil {
		return "", err
	}

	err = s.fileStorage.Write(ctx, s.metadataFile(fileName), metadataInBytes)
	if err != nil {
		s.fileStorage.Delete(ctx, fileName)
		return "", err
	}

	return fileName, nil
}

func (s *FileHostingService) scheduleDeleteFile(fileName string, sha1 string, expiredAt time.Time) error {
	msg := deleteFileMessage{
		FileName:  fileName,
		Sha1:      sha1,
		ExpiredAt: expiredAt,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return apperr.ErrInternalServerError.WithMessage("Failed to marshal delete message")
	}

	err = s.mq.Publish(fileDeletionQueueName, bytes)
	if err != nil {
		return apperr.ErrInternalServerError.WithMessage("Fail schedule file deletion")
	}

	return nil
}

func (s *FileHostingService) handleDeleteFileMessage(msg amqp.Delivery) {
	var delMsg deleteFileMessage
	if err := json.Unmarshal(msg.Body, &delMsg); err != nil {
		logging.L(s.ctx).Error("Failed to unmarshal delete file message", logging.ErrAttr(err))
		msg.Nack(false, false)
		return
	}

	if time.Now().Before(delMsg.ExpiredAt) {
		msg.Nack(false, true)
		return
	}

	metadata, err := s.GetFileMetadata(s.ctx, delMsg.FileName)
	if err != nil {
		logging.L(s.ctx).Warn("Failed to read metadata", logging.ErrAttr(err))
		msg.Ack(false)
		return
	}
	if metadata.Sha1 != delMsg.Sha1 {
		logging.L(s.ctx).Warn("SHA1 mismatch", logging.ErrAttr(err))
		msg.Nack(false, false)
		return
	}

	if time.Now().Before(metadata.ExpiredAt) {
		msg.Nack(false, true)
		return
	}

	err = s.fileStorage.Delete(s.ctx, delMsg.FileName)
	if err != nil {
		logging.L(s.ctx).Error("Failed to delete file", logging.ErrAttr(err))
		msg.Nack(false, true)
		return
	}

	err = s.fileStorage.Delete(s.ctx, s.metadataFile(delMsg.FileName))
	if err != nil {
		logging.L(s.ctx).Error("Failed to delete metadata file", logging.ErrAttr(err))
	}

	msg.Ack(false)

	logging.L(s.ctx).Info("Delete file", logging.StringAttr("file", delMsg.FileName))
}

func (s *FileHostingService) parseDuration(raw string, allowInfinite ...bool) time.Duration {
	isAllowedInfinite := false
	if len(allowInfinite) > 0 {
		isAllowedInfinite = allowInfinite[0]
	}

	if raw == "-1" && isAllowedInfinite {
		return 0
	} else if raw == "5m" {
		return 5 * time.Minute
	} else if raw == "30m" {
		return 30 * time.Minute
	} else if raw == "1h" || raw == "60m" {
		return time.Hour
	} else if raw == "1d" || raw == "24h" {
		return 24 * time.Hour
	} else if raw == "1w" || raw == "7d" {
		return 7 * 24 * time.Hour
	}

	return defaultFileDuration
}

func (s *FileHostingService) metadataFile(file string) string {
	return fmt.Sprintf("%s.metadata", file)
}

func (s *FileHostingService) generateFileName() string {
	now := time.Now().UnixMilli()

	timePart := strconv.FormatInt(now, 36)
	if len(timePart) > 4 {
		timePart = timePart[len(timePart)-4:]
	}

	randPart := fmt.Sprintf("%x", rand.Int32N(0x10000))
	return timePart + randPart
}

func (s *FileHostingService) sha1(data []byte) string {
	hash := sha1.Sum(data)
	return fmt.Sprintf("%x", hash)
}
