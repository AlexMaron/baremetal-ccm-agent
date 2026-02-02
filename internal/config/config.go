package config

import (
	"log"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env             string          `yaml:"env" env-default:"prod" validate:"oneof=prod dev test"`
	HTTPServer      HTTPServer      `yaml:"http_server" validate:"required"`
	Kubeconfig      string          `yaml:"kubeconfig" validate:"required"`
	Auth            Auth            `yaml:"auth" validate:"required"`
	HaproxyAuth     HaproxyAuth     `yaml:"haproxy_auth" validate:"required"`
}

type Auth struct {
	Username string `yaml:"username" validate:"required"`
	Password string `yaml:"password" validate:"required"`
}

type HaproxyAuth struct {
	Username string `yaml:"username" validate:"required"`
	Password string `yaml:"password" validate:"required"`
}

type HTTPServer struct {
	Address      string        `yaml:"address" env-default:"localhost:8082" validate:"required"`
	Timeout      time.Duration `yaml:"timeout" env-default:"4s"`
	Idle_timeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type Args struct {
	Config string `arg:"--config" help:"Path to config file."`
}

func MustLoad() *Config {
	var args Args
	arg.MustParse(&args)

	var cfg Config

	if err := cleanenv.ReadConfig(args.Config, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		log.Fatalf("invalid config: %s", err)
	}

	return &cfg
}
