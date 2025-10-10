package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Fail read config: %v", err)
	}

	cfg := newConfig(v)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func path(path ...string) string {
	return strings.Join(path, ".")
}
