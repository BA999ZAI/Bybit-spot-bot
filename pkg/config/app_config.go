package config

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

func LoadFromENV[Type any]() (Type, error) {
	var config Type
	err := envconfig.Process("", &config)
	return config, err
}

type BaseConfig struct {
	Version     string `envconfig:"VERSION" default:"dev"`
	Environment string `envconfig:"ENVIRONMENT" default:"local"`
	ServiceID   string `envconfig:"SERVICE_ID" default:"cryptorg-bot"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat   string `envconfig:"LOG_FORMAT" default:"json"`
}

func (c *BaseConfig) IsLocal() bool {
	return strings.ToLower(c.Environment) == "local"
}

func (c *BaseConfig) IsProduction() bool {
	env := strings.ToLower(c.Environment)
	return env == "production" || env == "prod"
}

func (c *BaseConfig) IsDevelopment() bool {
	env := strings.ToLower(c.Environment)
	return env == "development" || env == "dev" || env == "local"
}

type ServerConfig struct {
	Port         string `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout  int    `envconfig:"SERVER_READ_TIMEOUT" default:"30"`
	WriteTimeout int    `envconfig:"SERVER_WRITE_TIMEOUT" default:"30"`
	IdleTimeout  int    `envconfig:"SERVER_IDLE_TIMEOUT" default:"60"`
}

type BybitConfig struct {
	APIKey    string `envconfig:"BYBIT_API_KEY" required:"true"`
	SecretKey string `envconfig:"BYBIT_API_SECRET" required:"true"`
	Testnet   bool   `envconfig:"BYBIT_TESTNET" default:"false"`
	Symbol    string `envconfig:"SYMBOL" default:"SOLUSDT"`
}

type Config struct {
	Base   BaseConfig   `envconfig:""`
	Server ServerConfig `envconfig:""`
	Bybit  BybitConfig  `envconfig:""`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg, err := LoadFromENV[Config]()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) GetLoggerConfig() map[string]interface{} {
	return map[string]interface{}{
		"level":  c.Base.LogLevel,
		"format": c.Base.LogFormat,
	}
}
