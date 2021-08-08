package config

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/trinhdaiphuc/env_config"
	"time"
)

type AppConfig struct {
	Host           string        `env:"HOST,0.0.0.0"`
	Port           int           `env:"PORT,8080"`
	Separator      string        `env:"SEPARATOR,/"`
	UseTLS         bool          `env:"USE_TLS,false"`
	KeyFile        string        `env:"KEY_FILE"`
	CertFile       string        `env:"CERT_FILE"`
	CaFile         string        `env:"CA_FILE"`
	UseAuth        bool          `env:"USE_AUTH,true"`
	ConnectTimeout time.Duration `env:"CONNECT_TIMEOUT,5s"`
	SecretKey      []byte        `env:"SECRET_KEY,secret"`
	ExpiredTime    time.Duration `env:"EXPIRED_TIME,24h"`
}

var cfg = &AppConfig{}

func Load() {
	if err := env_config.EnvStruct(cfg); err != nil {
		panic(err)
	}
}

func GetConfig() *AppConfig {
	return cfg
}
