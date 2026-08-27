package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port       int
	Database   string
	JWTSecret  string
	JWTExpires int
	UploadDir  string
}

func Load() (Config, error) {
	port := 8080
	if value := os.Getenv("PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 {
			return Config{}, fmt.Errorf("invalid PORT: %q", value)
		}
		port = parsed
	}

	database := getenv("DATABASE_PATH", "./data/plant-diary.db")
	uploadDir := getenv("UPLOAD_DIR", "./uploads")
	if err := os.MkdirAll(filepath.Dir(database), 0755); err != nil {
		return Config{}, fmt.Errorf("create database directory: %w", err)
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return Config{}, fmt.Errorf("create upload directory: %w", err)
	}

	return Config{
		Port:       port,
		Database:   database,
		JWTSecret:  getenv("JWT_SECRET", "plant-diary-development-secret"),
		JWTExpires: 24,
		UploadDir:  uploadDir,
	}, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
