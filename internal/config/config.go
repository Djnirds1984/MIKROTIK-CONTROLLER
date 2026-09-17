package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port   int
	DBHost string
	DBPort int
	DBUser string
	DBPass string
	DBName string
}

func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName)
}

func Load() *Config {
	cfg := &Config{
		Port:   8080,
		DBHost: "localhost",
		DBPort: 5433,
		DBUser: "pisowifi",
		DBPass: "pisowifi",
		DBName: "pisowifi",
	}

	if p := os.Getenv("MIKROTIK_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			cfg.Port = v
		}
	}
	if v := os.Getenv("PISOWIFI_DB_HOST"); v != "" {
		cfg.DBHost = v
	}
	if p := os.Getenv("PISOWIFI_DB_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			cfg.DBPort = v
		}
	}
	if v := os.Getenv("PISOWIFI_DB_USER"); v != "" {
		cfg.DBUser = v
	}
	if v := os.Getenv("PISOWIFI_DB_PASSWORD"); v != "" {
		cfg.DBPass = v
	}
	if v := os.Getenv("PISOWIFI_DB_NAME"); v != "" {
		cfg.DBName = v
	}

	return cfg
}
