package main

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ApiKey        string `envconfig:"api_key" required:"true"`
	ServerPort    string `envconfig:"server_port" default:"8080"`
	RedisAddr     string `envconfig:"redis_addr" default:"localhost:6379"`
	RedisPassword string `envconfig:"redis_password" default:""`
	RedisDB       int    `envconfig:"redis_db" default:"0"`
}

func Read() (*Config, error) {
	_ = godotenv.Overload(".env", ".env.local")
	config := new(Config)
	if err := envconfig.Process("", config); err != nil {
		return nil, err
	}
	return config, nil
}
