package config

import (
	"os"
	"strconv"
)

// Config структура для хранения настроек
type Config struct {
	ServerAddress    string
	ServerPort       string
	MinioEndpoint    string
	AdminAccessKey   string
	AdminSecretKey   string
	MinioSecure      bool
	JWTSecret        string
	JWTRefreshSecret string
	JWTAccessTTL     int
	JWTRefreshTTL    int
	HostDB           string
	PortDB           string
	UserDB           string
	PasswordDB       string
	NameDB           string
}

// New возвращает новый экземпляр Config
func New() *Config {
	return &Config{
		ServerAddress:    getEnv("SERVER_ADDRESS", "localhost"),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		MinioEndpoint:    getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AdminAccessKey:   os.Getenv("MINIO_ADMIN_ACCESS"),
		AdminSecretKey:   os.Getenv("MINIO_ADMIN_SECRET"),
		MinioSecure:      getEnvBool("MINIO_SECURE", false),
		JWTSecret:        getEnv("JWT_SECRET", "jwt-secret"),
		JWTRefreshSecret: getEnv("JWT_REFRESH", "jwt-refresh-secret"),
		JWTAccessTTL:     getEnvInt("JWT_ACCESS_TTL", 15*60),
		JWTRefreshTTL:    getEnvInt("JWT_REFRESH_TTL", 7*24*60*60),
		HostDB:           getEnv("HOST_DB", ""),
		PortDB:           getEnv("PORT_DB", "5432"),
		UserDB:           getEnv("USER_DB", ""),
		PasswordDB:       getEnv("PASSWORD_DB", ""),
		NameDB:           getEnv("NAME_DB", ""),
	}
}

// getEnvInt возвращает значение переменной окружения в виде int или значение по умолчанию
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
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
