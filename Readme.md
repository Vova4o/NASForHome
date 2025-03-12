# **NAS Project**

Простой монолитный проект для создания локального NAS-сервера с использованием MinIO для хранения данных, Go для бэкенда и Next.js для фронтенда.

---

## **Архитектура**

Этот проект использует следующие компоненты:

- **MinIO** — объектное хранилище S3-совместимого типа для хранения файлов.
- **Go (Gin/CHI)** — бэкенд API для взаимодействия с MinIO и управления пользователями.
- **Next.js** — фронтенд для взаимодействия с сервером через Web UI.
- **PostgreSQL** — база данных для хранения метаданных (например, информация о файлах, пользователях).

Проект использует **Docker Compose** для упрощенного развертывания всех сервисов.

---

## **Стек технологий**

- **Backend**: Go Gin, MinIO (S3 API), PostgreSQL
- **Frontend**: Next.js (React)
- **Auth**: JWT
- **Security**: TLS, VPN (в будущем)
- **Storage**: MinIO / локальная файловая система (ext4, ZFS)

---

## **Установка и запуск**

1. **Клонируйте репозиторий:**

```bash
git clone https://github.com/vova4o/nasforhome
cd nasforhome
```

2. **Настройка Docker:**

Убедитесь, что у вас установлен **Docker** и **Docker Compose**. Затем запустите контейнеры:

```bash
docker-compose up
```

- **MinIO будет доступен на порту 9000 (API)** и 9001 (Web UI).
- **PostgreSQL** будет доступен на порту 5432.
  
   Для MinIO доступ по умолчанию:
   - Логин: `admin`
   - Пароль: `strongpassword`

3. **Запуск бэкенда (Go):**

Если бэкенд еще не запущен с помощью Docker, его можно запустить отдельно:

```bash
go run main.go
```

4. **Запуск фронтенда (Next.js):**

Для фронтенда, выполните:

```bash
cd frontend
npm install
npm run dev
```

Фронтенд будет доступен по адресу [http://localhost:3000](http://localhost:3000).

---

## **Подключение к MinIO через Go SDK**

Пример кода для подключения к MinIO и получения списка бакетов:

```go
package main

import (
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	// Подключаемся к MinIO
	minioClient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("admin", "strongpassword", ""),
		Secure: false, // если используем HTTP, иначе true для HTTPS
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Проверим доступность MinIO
	buckets, err := minioClient.ListBuckets(nil)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Buckets:")
	for _, bucket := range buckets {
		fmt.Println(bucket.Name)
	}
}
```

---

## **Авторизация (JWT)**

Для защиты API и доступа к данным используется **JWT** (JSON Web Token). В будущем можно добавить поддержку ролей и прав доступа для различных пользователей (например, админ/пользователь).

**Пример создания JWT в Go:**

```go
package auth

import (
	"time"
	"github.com/golang-jwt/jwt/v4"
)

func GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * 24).Unix(), // Токен истекает через 24 часа
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret"))
}
```

---

## **Доступ извне (TLS)**

Для подключения к серверу по **TLS** настройте соответствующие сертификаты для безопасного доступа:

1. Создайте SSL-сертификаты.
2. Настройте сервер Go для использования HTTPS.
   
Доступ будет ограничен через защищенное соединение по порту 443.

---

## **План на будущее**

- Добавление **VPN** (WireGuard/OpenVPN) для безопасного подключения к серверу извне.
- Реализация **файловых версий и дедупликации**.
- Возможность работы с **Samba** или **NFS** для совместимости с другими устройствами.

---

## **Лицензия**

Этот проект использует лицензию MIT. См. файл [LICENSE](LICENSE) для подробностей.

---
