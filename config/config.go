package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type LogConfig struct {
	Filename string    `mapstructure:"filename"`
	Level    log.Level `mapstructure:"level"`
}

type DatabaseConfig struct {
	// Host overrides DB_HOST for the PHP introspection process.
	// Useful when the DB is exposed on a non-default loopback address (e.g. Docker).
	Host string `mapstructure:"host"`
}

type Config struct {
	Log      LogConfig      `mapstructure:"log"`
	Database DatabaseConfig `mapstructure:"database"`
}

func Parse(v *viper.Viper) (Config, error) {
	var cfg Config
	err := v.Unmarshal(&cfg, viper.DecodeHook(LogLevelHook))
	return cfg, err
}
