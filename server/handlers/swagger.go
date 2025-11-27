package handlers

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	
	"httpserver/docs"
)

// SwaggerHandler обработчик для Swagger UI
type SwaggerHandler struct{}

// NewSwaggerHandler создает новый обработчик Swagger
func NewSwaggerHandler() *SwaggerHandler {
	return &SwaggerHandler{}
}

// RegisterSwaggerRoutes регистрирует маршруты Swagger в Gin роутере
func RegisterSwaggerRoutes(router *gin.Engine) {
	// Устанавливаем информацию о Swagger из сгенерированной документации
	docs.SwaggerInfo.Host = "localhost:9999"
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	
	// Регистрируем Swagger UI с использованием сгенерированной документации
	// Используем URL опцию для явного указания пути к doc.json
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))
}

