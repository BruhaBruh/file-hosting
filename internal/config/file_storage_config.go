package config

import (
	"errors"

	"github.com/spf13/viper"
)

type FileStorageConfig struct {
	basic *BasicFileStorageConfig
}

func newFileStorageConfig(prefix string, v *viper.Viper) *FileStorageConfig {
	return &FileStorageConfig{
		basic: newBasicFileStorageConfig(path(prefix, "basic"), v),
	}
}

func (c *FileStorageConfig) Basic() *BasicFileStorageConfig {
	return c.basic
}

func (c *FileStorageConfig) Validate() error {
	if c.basic.enabled {
		return c.basic.Validate()
	}
	return errors.New("one of file storage must be enabled")
}
