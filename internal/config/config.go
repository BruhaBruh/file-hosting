package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	origin      string
	port        int
	apiKey      string
	logger      *LoggerConfig
	fileStorage *FileStorageConfig
	rabbitmq    *RabbitMQConfig
}

func newConfig(v *viper.Viper) *Config {
	v.SetDefault("origin", "http://localhost:8080")
	v.SetDefault("port", 8080)
	v.SetDefault("API_KEY", "")

	return &Config{
		origin:      v.GetString("origin"),
		port:        v.GetInt("port"),
		apiKey:      v.GetString("API_KEY"),
		logger:      newLoggerConfig("logger", v),
		fileStorage: newFileStorageConfig("fileStorage", v),
		rabbitmq:    newRabbitMQConfig("rabbitmq", v),
	}
}

func (c *Config) Origin() string {
	return c.origin
}

func (c *Config) Port() int {
	return c.port
}

func (c *Config) ApiKey() string {
	return c.apiKey
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
	if c.port < 0 || c.port > 65535 {
		return fmt.Errorf("invalid port: %d", c.port)
	}

	if c.apiKey == "" {
		return errors.New("API_KEY is required")
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
