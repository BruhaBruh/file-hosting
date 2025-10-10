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
	"github.com/goccy/go-json"
)

type FileHostingService struct {
	fileStorage storage.FileStorage
}

func NewFileHostingService(fileStorage storage.FileStorage) *FileHostingService {
	return &FileHostingService{
		fileStorage: fileStorage,
	}
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

func (s *FileHostingService) UploadFile(ctx context.Context, data []byte, metadata *domain.FileMetadata) (string, error) {
	if s.fileStorage.IsExist(ctx, metadata.Name) {
		oldFileData, err := s.fileStorage.Read(ctx, metadata.Name)
		if err == nil {
			newSha1 := s.sha1(data)
			oldSha1 := s.sha1(oldFileData)
			if newSha1 == oldSha1 {
				return metadata.Name, nil
			}
		}

		now := time.Now().UnixNano()
		newFileName := fmt.Sprintf("%s.%d", metadata.Name, now)
		newMetadataFileName := fmt.Sprintf("%s.%d.metadata", metadata.Name, now)
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
		CreatedAt:  time.Now(),
		BackupName: metadata.BackupName,
	}

	metadataInBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return "", apperr.ErrInternalServerError.WithMessage("Fail serialize metadata")
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

func (s *FileHostingService) UploadFileWithGenerativeName(ctx context.Context, data []byte, metadata *domain.FileMetadata) (string, error) {
	fileName := s.generateFileName()
	for {
		if s.fileStorage.IsExist(ctx, fileName) {
			fileName = s.generateFileName()
			continue
		}
		break
	}

	newMetadata := &domain.FileMetadata{
		Name:       metadata.Name,
		MimeType:   metadata.MimeType,
		Sha1:       s.sha1(data),
		Meta:       metadata.Meta,
		CreatedAt:  time.Now(),
		BackupName: metadata.BackupName,
	}

	metadataInBytes, err := json.Marshal(newMetadata)
	if err != nil {
		return "", apperr.ErrInternalServerError.WithMessage("Fail serialize metadata")
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
