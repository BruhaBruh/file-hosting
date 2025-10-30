package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	origin      string
	apiKey      string
	http        *HTTPConfig
	grpc        *GRPCConfig
	logger      *LoggerConfig
	fileStorage *FileStorageConfig
	rabbitmq    *RabbitMQConfig
}

func newConfig(v *viper.Viper) *Config {
	v.SetDefault("origin", "http://localhost:8080")
	v.SetDefault("port", 8080)

	return &Config{
		origin:      v.GetString("origin"),
		apiKey:      v.GetString("API_KEY"),
		http:        newHTTPConfig("http", v),
		grpc:        newGRPCConfig("grpc", v),
		logger:      newLoggerConfig("logger", v),
		fileStorage: newFileStorageConfig("fileStorage", v),
		rabbitmq:    newRabbitMQConfig("rabbitmq", v),
	}
}

func (c *Config) Origin() string {
	return c.origin
}

func (c *Config) ApiKey() string {
	return c.apiKey
}

func (c *Config) HTTP() *HTTPConfig {
	return c.http
}

func (c *Config) GRPC() *GRPCConfig {
	return c.grpc
}

func (c *Config) Logger() *LoggerConfig {
	return c.logger
}

func (c *Config) FileStorage() *FileStorageConfig {
	return c.fileStorage
}

func (c *Config) RabbitMQ() *RabbitMQConfig {
	return c.rabbitmq
}

func (c *Config) Validate() error {
	if len(c.apiKey) == 0 {
		return errors.New("API_KEY is required")
	}

	if err := c.http.Validate(); err != nil {
		return fmt.Errorf("invalid http config: %w", err)
	}

	if err := c.grpc.Validate(); err != nil {
		return fmt.Errorf("invalid grpc config: %w", err)
	}

	if err := c.logger.Validate(); err != nil {
		return fmt.Errorf("invalid logger config: %w", err)
	}

	if err := c.fileStorage.Validate(); err != nil {
		return fmt.Errorf("invalid file storage config: %w", err)
	}

	if err := c.rabbitmq.Validate(); err != nil {
		return fmt.Errorf("invalid rabbitmq config: %w", err)
	}

	return nil
}
