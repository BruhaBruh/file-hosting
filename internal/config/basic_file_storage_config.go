package config

import (
	"errors"

	"github.com/spf13/viper"
)

type BasicFileStorageConfig struct {
	enabled   bool
	directory string
}

func newBasicFileStorageConfig(prefix string, v *viper.Viper) *BasicFileStorageConfig {
	v.SetDefault(path(prefix, "enabled"), true)
	v.SetDefault(path(prefix, "directory"), "files")

	return &BasicFileStorageConfig{
		enabled:   v.GetBool(path(prefix, "enabled")),
		directory: v.GetString(path(prefix, "directory")),
	}
}

func (c *BasicFileStorageConfig) Enabled() bool {
	return c.enabled
}

func (c *BasicFileStorageConfig) Directory() string {
	return c.directory
}

func (c *BasicFileStorageConfig) Validate() error {
	if c.directory == "" {
		return errors.New("directory cannot be empty")
	}
	return nil
}
