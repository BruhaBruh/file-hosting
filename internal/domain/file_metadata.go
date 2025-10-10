package domain

import (
	"time"

	"github.com/goccy/go-json"
)

type FileMetadata struct {
	Name       string              `json:"name"`
	MimeType   string              `json:"mime_type"`
	Sha1       string              `json:"sha1"`
	Meta       map[string][]string `json:"meta"`
	CreatedAt  time.Time           `json:"created_at"`
	ExpiredAt  time.Time           `json:"expired_at"`
	BackupName string              `json:"backup_name,omitempty"`
}

func NewFileMetadataFromBytes(data []byte) (*FileMetadata, error) {
	var metadata FileMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}
