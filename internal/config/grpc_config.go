package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type GRPCConfig struct {
	enabled bool
	port    int
}

func newGRPCConfig(prefix string, v *viper.Viper) *GRPCConfig {
	v.SetDefault(path(prefix, "enabled"), true)
	v.SetDefault(path(prefix, "port"), 8080)

	return &GRPCConfig{
		enabled: v.GetBool(path(prefix, "enabled")),
		port:    v.GetInt(path(prefix, "port")),
	}
}

func (c *GRPCConfig) Enabled() bool {
	return c.enabled
}

func (c *GRPCConfig) Port() int {
	return c.port
}

func (c *GRPCConfig) Validate() error {
	if c.enabled && (c.port < 0 || c.port > 65535) {
		return fmt.Errorf("invalid port: %d", c.port)
	}

	return nil
}
