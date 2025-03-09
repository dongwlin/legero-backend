package config

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	HttpServer HttpServerConfig `mapstructure:"http"`
	Log        LogConfig        `mapstructure:"log"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
}

type HttpServerConfig struct {
	Port int
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func New() *Config {

	conf := &Config{}

	conf.Load()

	return conf
}

func (c *Config) Load() {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to read config file")

	}

	if err := v.Unmarshal(c); err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to unmarshal config file")
	}
}
