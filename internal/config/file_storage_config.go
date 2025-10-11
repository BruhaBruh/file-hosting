package config

import (
	"errors"

	"github.com/spf13/viper"
)

type FileStorageConfig struct {
	basic *BasicFileStorageConfig
	s3    *S3FileStorageConfig
}

func newFileStorageConfig(prefix string, v *viper.Viper) *FileStorageConfig {
	return &FileStorageConfig{
		basic: newBasicFileStorageConfig(path(prefix, "basic"), v),
		s3:    newS3FileStorageConfig(path(prefix, "s3"), v),
	}
}

func (c *FileStorageConfig) Basic() *BasicFileStorageConfig {
	return c.basic
}

func (c *FileStorageConfig) S3() *S3FileStorageConfig {
	return c.s3
}

func (c *FileStorageConfig) Validate() error {
	enabledCount := 0
	if c.basic.enabled {
		enabledCount += 1
	}
	if c.s3.enabled {
		enabledCount += 1
	}
	if enabledCount > 1 {
		return errors.New("only one file storage can be enabled")
	}

	if c.basic.enabled {
		return c.basic.Validate()
	}
	if c.s3.enabled {
		return c.s3.Validate()
	}

	return errors.New("one of file storage must be enabled")
}
