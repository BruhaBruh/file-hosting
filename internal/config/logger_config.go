package config

import (
	"fmt"
	"strings"

	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/spf13/viper"
)

type LoggerConfig struct {
	level            string
	isJSON           bool
	addSource        bool
	filePath         string
	fileMaxSizeInMB  int
	fileMaxBackups   int
	fileMaxAgeInDays int
	fileCompress     bool
}

func newLoggerConfig(prefix string, v *viper.Viper) *LoggerConfig {
	v.SetDefault(path(prefix, "level"), "info")
	v.SetDefault(path(prefix, "format"), "json")
	v.SetDefault(path(prefix, "addSource"), true)
	v.SetDefault(path(prefix, "file.path"), "")
	v.SetDefault(path(prefix, "file.maxSizeInMB"), 10)
	v.SetDefault(path(prefix, "file.maxBackups"), 3)
	v.SetDefault(path(prefix, "file.maxAgeInDays"), 14)
	v.SetDefault(path(prefix, "file.compress"), false)

	return &LoggerConfig{
		level:            v.GetString(path(prefix, "level")),
		isJSON:           strings.ToLower(v.GetString(path(prefix, "format"))) == "json",
		addSource:        v.GetBool(path(prefix, "addSource")),
		filePath:         v.GetString(path(prefix, "file.path")),
		fileMaxSizeInMB:  v.GetInt(path(prefix, "file.maxSizeInMB")),
		fileMaxBackups:   v.GetInt(path(prefix, "file.maxBackups")),
		fileMaxAgeInDays: v.GetInt(path(prefix, "file.maxAgeInDays")),
		fileCompress:     v.GetBool(path(prefix, "file.compress")),
	}
}

func (c *LoggerConfig) Level() string {
	return c.level
}

func (c *LoggerConfig) IsJSON() bool {
	return c.isJSON
}

func (c *LoggerConfig) AddSource() bool {
	return c.addSource
}

func (c *LoggerConfig) FilePath() string {
	return c.filePath
}

func (c *LoggerConfig) FileMaxSizeInMB() int {
	return c.fileMaxSizeInMB
}

func (c *LoggerConfig) FileMaxBackups() int {
	return c.fileMaxBackups
}

func (c *LoggerConfig) FileMaxAgeInDays() int {
	return c.fileMaxAgeInDays
}

func (c *LoggerConfig) FileCompress() bool {
	return c.fileCompress
}

func (c *LoggerConfig) Validate() error {
	if c.fileMaxSizeInMB < 0 {
		return fmt.Errorf("invalid file max size in MB: %d", c.fileMaxSizeInMB)
	}

	if c.fileMaxBackups < 0 {
		return fmt.Errorf("invalid file max backups: %d", c.fileMaxBackups)
	}

	if c.fileMaxAgeInDays < 0 {
		return fmt.Errorf("invalid file max age in days: %d", c.fileMaxAgeInDays)
	}

	return nil
}

func (c *LoggerConfig) Build() *logging.Logger {
	return logging.New(
		logging.WithLevel(c.Level()),
		logging.WithIsJSON(c.IsJSON()),
		logging.WithAddSource(c.AddSource()),
		logging.WithLogFilePath(c.FilePath()),
		logging.WithLogFileMaxSizeMB(c.FileMaxSizeInMB()),
		logging.WithLogFileMaxBackups(c.FileMaxBackups()),
		logging.WithLogFileMaxAgeDays(c.FileMaxAgeInDays()),
		logging.WithLogFileCompress(c.FileCompress()),
	)
}
