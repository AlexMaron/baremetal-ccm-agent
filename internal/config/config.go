package config

import (
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env               string        `env:"ENV" env-default:"prod" validate:"oneof=prod dev test"`
	HTTPAddress       string        `env:"HTTP_ADDRESS" env-default:"0.0.0.0:8080" validate:"required"`
	HTTPTimeout       time.Duration `env:"HTTP_TIMEOUT" env-default:"4s"`
	HTTPIdleTimeout   time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	Kubeconfig        string        `env:"KUBECONFIG"`
	DataPlaneHosts    []string      `env:"DATA_PLANE_HOSTS" validate:"required"`
	Username          string        `env:"USERNAME" validate:"required"`
	Password          string        `env:"PASSWORD" validate:"required"`
	DataPlaneUsername string        `env:"DATA_PLANE_USERNAME" validate:"required"`
	DataPlanePassword string        `env:"DATA_PLANE_PASSWORD" validate:"required"`
	ExternaIP         string        `env:"EXTERNAL_IP" validate:"required"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, err
	}

	for _, h := range cfg.DataPlaneHosts {
		if strings.TrimSpace(h) == "" {
			log.Fatalf("DATA_PLANE_HOSTS contains empty host")
		}
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()

	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	return cfg
}
