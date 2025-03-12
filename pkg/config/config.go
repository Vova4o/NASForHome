package config

import (
	"os"
	"strconv"
)

// Config структура для хранения настроек
type Config struct {
	MinioEndpoint string
	MinioUser     string
	MinioPassword string
	MinioSecure   bool
}

// New возвращает новый экземпляр Config
func New() *Config {
	return &Config{
		MinioEndpoint: getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioUser:     getEnv("MINIO_USER", "minio"),
		MinioPassword: getEnv("MINIO_PASSWORD", "minio123"),
		MinioSecure:   getEnvBool("MINIO_SECURE", false),
	}
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvBool возвращает значение переменной окружения в виде bool или значение по умолчанию
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return result
}
