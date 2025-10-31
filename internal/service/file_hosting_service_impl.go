package service

import (
	"context"
	"crypto/sha1"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/internal/domain"
	"github.com/bruhabruh/file-hosting/internal/storage"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/bruhabruh/file-hosting/pkg/rabbitmq"
	"github.com/goccy/go-json"
	"github.com/streadway/amqp"
)

const fileDeletionQueueName = "file-hosting-service/delete-file"

var infiniteTimeStamp = time.Unix(0, 0)

type deleteFileMessage struct {
	FileName  string    `json:"fileName"`
	Sha1      string    `json:"sha1"`
	ExpiredAt time.Time `json:"expiredAt"`
}

type FileHostingServiceImpl struct {
	ctx         context.Context
	fileStorage storage.FileStorage
	mq          *rabbitmq.RabbitMQ
}

func NewFileHostingService(ctx context.Context, fileStorage storage.FileStorage, mq *rabbitmq.RabbitMQ) (FileHostingService, error) {
	service := &FileHostingServiceImpl{
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

func (s *FileHostingServiceImpl) GetFiles(ctx context.Context) ([]*domain.FileMetadata, error) {
	fileNames, err := s.fileStorage.Files(ctx)
	if err != nil {
		return nil, err
	}

	files := make([]*domain.FileMetadata, len(fileNames))
	for i, fileName := range fileNames {
		file, err := s.GetFileMetadata(ctx, fileName)
		if err != nil {
			return nil, err
		}
		files[i] = file
	}

	return files, nil
}

func (s *FileHostingServiceImpl) GetFile(ctx context.Context, file string) (*domain.File, error) {
	data, err := s.fileStorage.Read(ctx, file)
	if err != nil {
		return nil, err
	}

	metadata, err := s.GetFileMetadata(ctx, file)
	if err != nil {
		return nil, err
	}

	return &domain.File{
		Content:  data,
		Metadata: metadata,
	}, nil
}

func (s *FileHostingServiceImpl) GetFileMetadata(ctx context.Context, file string) (*domain.FileMetadata, error) {
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

func (s *FileHostingServiceImpl) UploadFile(ctx context.Context, content []byte, metadata *domain.FileMetadata, rawDuration string) (string, *domain.File, error) {
	if strings.Contains(metadata.Name, "/") {
		return "", nil, apperr.ErrBadRequest.WithMessage("File name cannot contain '/'")
	}

	now := time.Now()

	metadata.UpdateContentType(content)

	expiredAt := infiniteTimeStamp
	if duration := parseDuration(rawDuration, true); duration != 0 {
		expiredAt = now.Add(duration)
	}

	if s.fileStorage.IsExist(ctx, metadata.Name) {
		oldFileData, err := s.fileStorage.Read(ctx, metadata.Name)
		if err == nil {
			newSha1 := s.sha1(content)
			oldSha1 := s.sha1(oldFileData)
			if newSha1 == oldSha1 {
				return metadata.Name, nil, nil
			}
		}
		newFileName := fmt.Sprintf("%s.%d", metadata.Name, now.UnixNano())
		newMetadataFileName := fmt.Sprintf("%s.%d.metadata", metadata.Name, now.UnixNano())

		oldMetadata, _ := s.GetFileMetadata(ctx, metadata.Name)
		if oldMetadata != nil && oldMetadata.ExpiredAt != infiniteTimeStamp {
			err = s.scheduleDeleteFile(newFileName, oldMetadata.Sha1, oldMetadata.ExpiredAt)
			if err != nil {
				return "", nil, err
			}
		}

		newMetadata := &domain.FileMetadata{
			Id:         newFileName,
			Name:       oldMetadata.Name,
			MimeType:   oldMetadata.MimeType,
			Sha1:       oldMetadata.Sha1,
			Meta:       oldMetadata.Meta,
			CreatedAt:  oldMetadata.CreatedAt,
			ExpiredAt:  oldMetadata.ExpiredAt,
			BackupName: oldMetadata.BackupName,
		}

		if err := s.fileStorage.Move(ctx, metadata.Name, newFileName); err != nil {
			return "", nil, err
		}
		if s.fileStorage.IsExist(ctx, s.metadataFile(metadata.Name)) {
			err = s.fileStorage.Delete(ctx, s.metadataFile(metadata.Name))
			if err != nil {
				return "", nil, err
			}

			metadataInBytes, err := json.Marshal(newMetadata)
			if err != nil {
				return "", nil, apperr.ErrInternalServerError.WithMessage("Fail serialize old metadata")
			}
			err = s.fileStorage.Write(ctx, newMetadataFileName, metadataInBytes, "application/json")
			if err != nil {
				return "", nil, err
			}
		}
		metadata.BackupName = newFileName
	}

	newMetadata := &domain.FileMetadata{
		Id:         metadata.Name,
		Name:       metadata.Name,
		MimeType:   metadata.MimeType,
		Sha1:       s.sha1(content),
		Meta:       metadata.Meta,
		CreatedAt:  now,
		ExpiredAt:  expiredAt,
		BackupName: metadata.BackupName,
	}

	metadataInBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return "", nil, apperr.ErrInternalServerError.WithMessage("Fail serialize metadata")
	}

	if newMetadata.ExpiredAt != infiniteTimeStamp {
		err = s.scheduleDeleteFile(newMetadata.Name, newMetadata.Sha1, newMetadata.ExpiredAt)
		if err != nil {
			return "", nil, err
		}
	}

	err = s.fileStorage.Write(ctx, newMetadata.Name, content, newMetadata.MimeType)
	if err != nil {
		return "", nil, err
	}

	err = s.fileStorage.Write(ctx, s.metadataFile(newMetadata.Name), metadataInBytes, "application/json")
	if err != nil {
		s.fileStorage.Delete(ctx, newMetadata.Name)
		return "", nil, err
	}

	return newMetadata.Name, &domain.File{
		Content:  content,
		Metadata: newMetadata,
	}, nil
}

func (s *FileHostingServiceImpl) UploadFileWithGenerativeName(ctx context.Context, content []byte, metadata *domain.FileMetadata, rawDuration string) (string, *domain.File, error) {
	fileName := s.generateFileName()
	for {
		if s.fileStorage.IsExist(ctx, fileName) {
			fileName = s.generateFileName()
			continue
		}
		break
	}

	now := time.Now()
	expiredAt := now.Add(parseDuration(rawDuration))

	metadata.UpdateContentType(content)

	newMetadata := &domain.FileMetadata{
		Id:         fileName,
		Name:       metadata.Name,
		MimeType:   metadata.MimeType,
		Sha1:       s.sha1(content),
		Meta:       metadata.Meta,
		CreatedAt:  now,
		ExpiredAt:  expiredAt,
		BackupName: metadata.BackupName,
	}

	metadataInBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return "", nil, apperr.ErrInternalServerError.WithMessage("Fail serialize metadata")
	}

	err = s.scheduleDeleteFile(fileName, newMetadata.Sha1, newMetadata.ExpiredAt)
	if err != nil {
		return "", nil, err
	}

	err = s.fileStorage.Write(ctx, fileName, content, newMetadata.MimeType)
	if err != nil {
		return "", nil, err
	}

	err = s.fileStorage.Write(ctx, s.metadataFile(fileName), metadataInBytes, "application/json")
	if err != nil {
		s.fileStorage.Delete(ctx, fileName)
		return "", nil, err
	}

	return fileName, &domain.File{
		Content:  content,
		Metadata: newMetadata,
	}, nil
}

func (s *FileHostingServiceImpl) scheduleDeleteFile(fileName string, sha1 string, expiredAt time.Time) error {
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

func (s *FileHostingServiceImpl) handleDeleteFileMessage(msg amqp.Delivery) {
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

func (s *FileHostingServiceImpl) metadataFile(file string) string {
	return fmt.Sprintf("%s.metadata", file)
}

func (s *FileHostingServiceImpl) generateFileName() string {
	now := time.Now().UnixMilli()

	timePart := strconv.FormatInt(now, 36)
	if len(timePart) > 4 {
		timePart = timePart[len(timePart)-4:]
	}

	randPart := fmt.Sprintf("%x", rand.Int32N(0x10000))
	return timePart + randPart
}

func (s *FileHostingServiceImpl) sha1(data []byte) string {
	hash := sha1.Sum(data)
	return fmt.Sprintf("%x", hash)
}
