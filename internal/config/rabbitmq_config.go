package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type RabbitMQConfig struct {
	host     string
	port     int
	username string
	password string
}

func newRabbitMQConfig(prefix string, v *viper.Viper) *RabbitMQConfig {
	v.SetDefault(path(prefix, "host"), "localhost")
	v.SetDefault(path(prefix, "port"), 5672)

	return &RabbitMQConfig{
		host:     v.GetString(path(prefix, "host")),
		port:     v.GetInt(path(prefix, "port")),
		username: v.GetString("RABBITMQ_USERNAME"),
		password: v.GetString("RABBITMQ_PASSWORD"),
	}
}

func (c *RabbitMQConfig) Host() string {
	return c.host
}

func (c *RabbitMQConfig) Port() int {
	return c.port
}

func (c *RabbitMQConfig) Username() string {
	return c.username
}

func (c *RabbitMQConfig) Password() string {
	return c.password
}

func (c *RabbitMQConfig) URL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d", c.username, c.password, c.host, c.port)
}

func (c *RabbitMQConfig) Validate() error {
	if c.host == "" {
		return fmt.Errorf("invalid host: %s", c.host)
	}

	if c.port < 0 || c.port > 65535 {
		return fmt.Errorf("invalid port: %d", c.port)
	}

	if c.username == "" {
		return fmt.Errorf("invalid username: %s", c.username)
	}

	if c.password == "" {
		return fmt.Errorf("invalid password: %s", c.password)
	}

	return nil
}
