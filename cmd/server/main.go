package main

import (
	"context"
	"log"

	"github.com.Vova4o/nasforhome/pkg/config"
	miniolocal "github.com.Vova4o/nasforhome/pkg/minio"
	"github.com/joho/godotenv"
)

func main() {
	// Пытаемся загрузить .env файл, но не прерываем выполнение, если его нет
	_ = godotenv.Load() // Игнорируем ошибку

	ctx := context.Background()

	config := config.New()

	minioClient, err := miniolocal.New(
		ctx,
		config.MinioEndpoint,
		config.MinioUser,
		config.MinioPassword,
		config.MinioSecure)
	if err != nil {
		log.Fatalf("ошибка создания клиента MinIO: %v", err)
	}

	buckets, err := minioClient.ListBuckets(ctx)
	log.Printf("Список бакетов: %v", buckets)
}
