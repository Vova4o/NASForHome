package main

import (
	"context"
	"log"

	apiv1 "github.com.Vova4o/nasforhome/internal/apiV1"
	"github.com.Vova4o/nasforhome/internal/service"
	"github.com.Vova4o/nasforhome/pkg/config"
	miniolocal "github.com.Vova4o/nasforhome/pkg/minio"
	"github.com.Vova4o/nasforhome/pkg/storagedb"
	"github.com/joho/godotenv"
)

func main() {
	// Пытаемся загрузить .env файл, но не прерываем выполнение, если его нет
	// Это позволяет использовать переменные окружения по умолчанию в случае отсутствия .env файла
	// Удобно для использования в продакшене без .env файла
	_ = godotenv.Load() // Игнорируем ошибку

	ctx := context.Background()

	config := config.New()

	minioAdmin, err := miniolocal.NewAdminClient(
		ctx,
		config.MinioEndpoint,
		config.AdminAccessKey,
		config.AdminSecretKey,
		config.MinioSecure)
	if err != nil {
		log.Fatalf("ошибка создания админского клиента MinIO: %v", err)
	}

	buckets, err := minioAdmin.Client.ListBuckets(ctx)
	if err != nil {
		log.Fatalf("ошибка получения списка бакетов: %v", err)
	}
	log.Printf("Список бакетов: %v", buckets)

	postgresDB, err := storagedb.New(
		config.HostDB,
		config.PortDB,
		config.UserDB,
		config.PasswordDB,
		config.NameDB)
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	err = postgresDB.InitDB()
	if err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}

	migrationVersion, err := postgresDB.GetCurrentDBVersion()
	if err != nil {
		log.Fatalf("Ошибка получения текущей версии базы данных: %v", err)
	}
	log.Printf("Текущая версия базы данных: %d", migrationVersion)

	service := service.New(postgresDB, minioAdmin, service.MinioConfig{
		Endpoint: config.MinioEndpoint,
		Secure:   config.MinioSecure,
	},
		service.JWTConfig{
			AccessSecret:  config.JWTSecret,
			RefreshSecret: config.JWTRefreshSecret,
			AccessTTL:     config.JWTAccessTTL,
			RefreshTTL:    config.JWTRefreshTTL,
		},
	)

	serverAddress := config.ServerAddress + ":" + config.ServerPort

	api := apiv1.New(service)
	if err := api.Run(serverAddress); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
