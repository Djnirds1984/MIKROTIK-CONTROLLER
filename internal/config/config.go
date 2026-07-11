package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port  int
	DBPath string
}

func Load() *Config {
	port := 8080
	if p := os.Getenv("MIKROTIK_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	dbPath := "mikrotik_controller.db"
	if p := os.Getenv("MIKROTIK_DB_PATH"); p != "" {
		dbPath = p
	}

	return &Config{
		Port:  port,
		DBPath: dbPath,
	}
}
