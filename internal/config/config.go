package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env        string `yaml:"env" env-default:"local"`
	HTTPServer `yaml:"http_server"`
	PostgreSQL PostgreConfig `yaml:"postgresql"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8081"`
	Timeout     time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type PostgreConfig struct {
	Host     string `yaml:"host" env-default:"localhost"`
	Port     int    `yaml:"port" env-default:"5432"`
	User     string `yaml:"user" env-default:"postgres"`
	Password string `yaml:"password" env-default:"postgres"`
	DBName   string `yaml:"dbname" env-default:"postgres"`
	SSLMode  string `yaml:"sslmode" env-default:"disable"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/local.yaml"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("error reading config file: %s", err)
	}

	return &cfg
}
