package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type HTTPConfig struct {
	enabled         bool
	allowPage       bool
	maxBodySizeInMB int
	port            int
}

func newHTTPConfig(prefix string, v *viper.Viper) *HTTPConfig {
	v.SetDefault(path(prefix, "enabled"), true)
	v.SetDefault(path(prefix, "allowPage"), true)
	v.SetDefault(path(prefix, "maxBodySizeInMB"), 10)
	v.SetDefault(path(prefix, "port"), 8080)

	return &HTTPConfig{
		enabled:         v.GetBool(path(prefix, "enabled")),
		allowPage:       v.GetBool(path(prefix, "allowPage")),
		maxBodySizeInMB: v.GetInt(path(prefix, "maxBodySizeInMB")),
		port:            v.GetInt(path(prefix, "port")),
	}
}

func (c *HTTPConfig) Enabled() bool {
	return c.enabled
}

func (c *HTTPConfig) AllowPage() bool {
	return c.allowPage
}

func (c *HTTPConfig) MaxBodySizeInMB() int {
	return c.maxBodySizeInMB
}

func (c *HTTPConfig) Port() int {
	return c.port
}

func (c *HTTPConfig) Validate() error {
	if c.enabled && (c.port < 0 || c.port > 65535) {
		return fmt.Errorf("invalid port: %d", c.port)
	}

	return nil
}
