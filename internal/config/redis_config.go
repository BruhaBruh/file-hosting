package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type RedisConfig struct {
	host     string
	port     int
	password string
	database int
}

func newRedisConfig(prefix string, v *viper.Viper) *RedisConfig {
	v.SetDefault(path(prefix, "host"), "localhost")
	v.SetDefault(path(prefix, "port"), 6379)
	v.SetDefault(path(prefix, "database"), 0)

	return &RedisConfig{
		host:     v.GetString(path(prefix, "host")),
		port:     v.GetInt(path(prefix, "port")),
		password: v.GetString("REDIS_PASSWORD"),
		database: v.GetInt(path(prefix, "database")),
	}
}

func (c *RedisConfig) Host() string {
	return c.host
}

func (c *RedisConfig) Port() int {
	return c.port
}

func (c *RedisConfig) Password() string {
	return c.password
}

func (c *RedisConfig) Database() int {
	return c.database
}

func (c *RedisConfig) URL() string {
	return fmt.Sprintf("%s:%d", c.host, c.port)
}

func (c *RedisConfig) Validate() error {
	if c.host == "" {
		return fmt.Errorf("invalid host: %s", c.host)
	}

	if c.port < 0 || c.port > 65535 {
		return fmt.Errorf("invalid port: %d", c.port)
	}

	return nil
}
