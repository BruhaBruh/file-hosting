package config

import (
	"errors"

	"github.com/spf13/viper"
)

type S3FileStorageConfig struct {
	enabled   bool
	endpoint  string
	region    string
	bucket    string
	directory string
	accessKey string
	secretKey string
	useSSL    bool
}

func newS3FileStorageConfig(prefix string, v *viper.Viper) *S3FileStorageConfig {
	v.SetDefault(path(prefix, "enabled"), false)
	v.SetDefault(path(prefix, "useSSL"), true)
	v.SetDefault(path(prefix, "directory"), "")

	return &S3FileStorageConfig{
		enabled:   v.GetBool(path(prefix, "enabled")),
		endpoint:  v.GetString(path(prefix, "endpoint")),
		region:    v.GetString(path(prefix, "region")),
		bucket:    v.GetString(path(prefix, "bucket")),
		directory: v.GetString(path(prefix, "directory")),
		accessKey: v.GetString("S3_ACCESS_KEY"),
		secretKey: v.GetString("S3_SECRET_KEY"),
		useSSL:    v.GetBool(path(prefix, "useSSL")),
	}
}

func (c *S3FileStorageConfig) Enabled() bool {
	return c.enabled
}

func (c *S3FileStorageConfig) Endpoint() string {
	return c.endpoint
}

func (c *S3FileStorageConfig) Region() string {
	return c.region
}

func (c *S3FileStorageConfig) Bucket() string {
	return c.bucket
}

func (c *S3FileStorageConfig) Directory() string {
	return c.directory
}

func (c *S3FileStorageConfig) AccessKey() string {
	return c.accessKey
}

func (c *S3FileStorageConfig) SecretKey() string {
	return c.secretKey
}

func (c *S3FileStorageConfig) UseSSL() bool {
	return c.useSSL
}

func (c *S3FileStorageConfig) Validate() error {
	if c.endpoint == "" {
		return errors.New("endpoint cannot be empty")
	}
	if c.region == "" {
		return errors.New("region cannot be empty")
	}
	if c.bucket == "" {
		return errors.New("bucket cannot be empty")
	}
	if c.accessKey == "" {
		return errors.New("access key cannot be empty")
	}
	if c.secretKey == "" {
		return errors.New("secret key cannot be empty")
	}
	return nil
}
