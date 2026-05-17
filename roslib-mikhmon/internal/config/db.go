package config

import (
	"fmt"
	"os"
	"strconv"
)

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

func LoadDBFromEnv() *DBConfig {
	port := 5432
	if v := os.Getenv("DB_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}
	return &DBConfig{
		Host:     getenv("DB_HOST", "localhost"),
		Port:     port,
		Name:     os.Getenv("DB_NAME"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		SSLMode:  getenv("DB_SSLMODE", "disable"),
	}
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}
