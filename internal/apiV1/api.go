package apiv1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com.Vova4o/nasforhome/internal/service"
	"github.com/gin-gonic/gin"
)

// APIV1 структура для работы с API v1
type APIV1 struct {
	router  *gin.Engine
	service *service.Service
}

// Config конфигурация для API
type Config struct {
	Service *service.Service
}

// New создает и настраивает экземпляр API v1
func New(service *service.Service) *APIV1 {
	router := gin.Default()

	api := &APIV1{
		router:  router,
		service: service,
	}

	api.setupRoutes()
	return api
}

// setupRoutes настраивает маршруты API
func (a *APIV1) setupRoutes() {
	// Создаем группу маршрутов с префиксом /api/v1
	v1 := a.router.Group("/api/v1")
	{
		// Публичные маршруты (без авторизации)
		v1.POST("/users/register", a.RegisterUser)
		v1.POST("/users/login", a.LoginUser)
		v1.POST("/users/refresh", a.RefreshToken)
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Защищенные маршруты с проверкой авторизации
		authorized := v1.Group("/")
		authorized.Use(a.authMiddleware())
		{
			// Маршруты для пользователя
			authorized.GET("/users/me", a.GetUserInfo)

			// Маршруты для файлов
			files := authorized.Group("/files")
			{
				files.GET("/list", a.ListFiles)
				files.GET("/download/:filename", a.DownloadFile)
				files.POST("/upload", a.UploadFile)
				files.DELETE("/:filename", a.DeleteFile)
			}

			// Маршруты для папок
			folders := authorized.Group("/folders")
			{
				folders.GET("/list", a.ListFolders)
				folders.POST("/create", a.CreateFolder)
				folders.DELETE("/:foldername", a.DeleteFolder)
			}
		}
	}
}

// authMiddleware возвращает middleware для проверки авторизации
func (a *APIV1) authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "отсутствует токен авторизации"})
            c.Abort()
            return
        }

        // Проверяем формат токена (Bearer token)
        tokenParts := strings.Split(authHeader, " ")
        if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "неверный формат токена"})
            c.Abort()
            return
        }

        // Проверяем валидность токена
        claims, err := a.service.VerifyAccessToken(tokenParts[1])
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "недействительный токен"})
            c.Abort()
            return
        }

        // Устанавливаем ID пользователя в контекст
        c.Set("userID", claims.UserID)
        c.Next()
    }
}

// RefreshToken обработчик для обновления токенов
func (a *APIV1) RefreshToken(c *gin.Context) {
    refreshToken, err := c.Cookie("refresh_token")
    if err != nil || refreshToken == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "отсутствует refresh токен"})
        return
    }

    // Обновляем токены
    tokens, err := a.service.RefreshTokens(refreshToken)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "недействительный refresh токен"})
        return
    }

    // Set new refresh token in an HTTP-only Secure cookie
    c.SetCookie("refresh_token", tokens.RefreshToken, tokens.RefreshTTL, "/", "", true, true)

    // Return new access token in JSON
    c.JSON(http.StatusOK, gin.H{
        "access_token": tokens.AccessToken,
        "expires_in":   tokens.ExpiresIn,
    })
}

// Run запускает сервер API
func (a *APIV1) Run(addr string) error {
	return a.router.Run(addr)
}

// RegisterUser обработчик для регистрации пользователя
func (a *APIV1) RegisterUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := a.service.RegisterUser(c.Request.Context(), req.Username, req.Password, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set refresh token as an HTTP-only, Secure cookie
	c.SetCookie("refresh_token", tokens.RefreshToken, 60*60*24*7, "/", "", true, true) // 7 days

	// Return access token in JSON response (Frontend will store it in memory)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Пользователь успешно зарегистрирован",
		"user_id":      user.ID,
		"access_token": tokens.AccessToken,
		"expires_in":   tokens.ExpiresIn,
	})
}

// LoginUser обработчик для входа пользователя
func (a *APIV1) LoginUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := a.service.LoginUser(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "неверное имя пользователя или пароль"})
		return
	}

	// Set refresh token as an HTTP-only, Secure cookie
	c.SetCookie("refresh_token", tokens.RefreshToken, tokens.RefreshTTL, "/", "", true, true) // 7 days

	// Return access token in JSON response (Frontend will store it in memory)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Пользователь успешно зарегистрирован",
		"user_id":      user.ID,
		"access_token": tokens.AccessToken,
		"expires_in":   tokens.ExpiresIn,
	})
}

// GetUserInfo обработчик для получения информации о пользователе
func (a *APIV1) GetUserInfo(c *gin.Context) {
	userID := c.GetInt("userID") // Получаем ID из контекста, установленного middleware

	user, err := a.service.Storagedb.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения данных пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.UserName,
		"email":    user.Email,
	})
}

// ListFiles обработчик для получения списка файлов
func (a *APIV1) ListFiles(c *gin.Context) {
	userID := c.GetInt("userID")
	prefix := c.DefaultQuery("prefix", "")
	recursive := c.DefaultQuery("recursive", "false") == "true"

	// Используем метод сервиса для получения списка файлов
	objects, err := a.service.ListUserFiles(c.Request.Context(), userID, prefix, recursive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения списка файлов"})
		return
	}

	// Форматируем результат для ответа
	var files []map[string]interface{}
	for _, obj := range objects {
		files = append(files, map[string]interface{}{
			"name":          obj.Key,
			"size":          obj.Size,
			"etag":          obj.ETag,
			"content_type":  obj.ContentType,
			"last_modified": obj.LastModified,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
	})
}

// DownloadFile обработчик для скачивания файла
func (a *APIV1) DownloadFile(c *gin.Context) {
	userID := c.GetInt("userID")
	filename := c.Param("filename")

	// Получаем файл с помощью сервиса
	object, stat, err := a.service.GetUserFile(c.Request.Context(), userID, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения файла"})
		return
	}

	// Устанавливаем заголовки
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", stat.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))

	// Передаем файл клиенту
	c.DataFromReader(http.StatusOK, stat.Size, stat.ContentType, object, nil)
}

// UploadFile обработчик для загрузки файла
func (a *APIV1) UploadFile(c *gin.Context) {
	userID := c.GetInt("userID")

	// Получаем загруженный файл из формы
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл не найден в запросе"})
		return
	}
	defer file.Close()

	// Получаем опциональный путь к папке
	path := c.DefaultPostForm("path", "")
	if path != "" && !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	filename := header.Filename
	objectName := path + filename

	// Определяем Content-Type файла
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Загружаем файл с помощью сервиса
	info, err := a.service.UploadUserFile(c.Request.Context(), userID, objectName, file, header.Size, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка загрузки файла"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "файл успешно загружен",
		"object_name": info.Key,
		"etag":        info.ETag,
		"size":        info.Size,
	})
}

// DeleteFile обработчик для удаления файла
func (a *APIV1) DeleteFile(c *gin.Context) {
	userID := c.GetInt("userID")
	filename := c.Param("filename")

	// Удаляем файл с помощью сервиса
	if err := a.service.DeleteUserFile(c.Request.Context(), userID, filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка удаления файла"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "файл успешно удален",
		"filename": filename,
	})
}

// ListFolders обработчик для получения списка папок
func (a *APIV1) ListFolders(c *gin.Context) {
	userID := c.GetInt("userID")
	prefix := c.DefaultQuery("prefix", "")

	// Используем метод сервиса для получения списка папок
	folders, err := a.service.ListUserFolders(c.Request.Context(), userID, prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения списка папок"})
		return
	}

	// Форматируем результат для ответа
	var result []map[string]interface{}
	for _, folder := range folders {
		result = append(result, map[string]interface{}{
			"name": folder,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"folders": result,
	})
}

// CreateFolder обработчик для создания папки
func (a *APIV1) CreateFolder(c *gin.Context) {
	userID := c.GetInt("userID")

	var req struct {
		FolderName string `json:"folder_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Создаем папку с помощью сервиса
	err := a.service.CreateUserFolder(c.Request.Context(), userID, req.FolderName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка создания папки"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "папка успешно создана",
		"foldername": req.FolderName,
	})
}

// DeleteFolder обработчик для удаления папки
func (a *APIV1) DeleteFolder(c *gin.Context) {
	userID := c.GetInt("userID")
	folderName := c.Param("foldername")

	// Удаляем папку с помощью сервиса
	if err := a.service.DeleteUserFolder(c.Request.Context(), userID, folderName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка удаления папки"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "папка успешно удалена",
		"foldername": folderName,
	})
}
