package domain

import (
	"github.com/goccy/go-json"
)

type File struct {
	Content  []byte        `json:"content"`
	Metadata *FileMetadata `json:"metadata"`
}

func NewFileFromBytes(data []byte) (*File, error) {
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return &file, nil
}
