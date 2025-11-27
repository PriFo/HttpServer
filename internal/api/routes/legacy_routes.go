package routes

import (
	"log"

	"github.com/gin-gonic/gin"
)

// LegacyRouteBinder описывает сущность, которая знает, как регистрировать legacy-маршруты.
type LegacyRouteBinder interface {
	RegisterLegacy(group *gin.RouterGroup)
}

// RegisterLegacyRoutes создает группу /api и делегирует регистрацию binder'у.
func RegisterLegacyRoutes(router *gin.Engine, binder LegacyRouteBinder) {
	if router == nil {
		log.Printf("[RegisterLegacyRoutes] ⚠ router is nil")
		return
	}
	if binder == nil {
		log.Printf("[RegisterLegacyRoutes] ⚠ binder is nil")
		return
	}

	log.Printf("[RegisterLegacyRoutes] Регистрация legacy routes в /api")
	
	// Регистрируем legacy routes напрямую в /api для обратной совместимости
	api := router.Group("/api")
	binder.RegisterLegacy(api)
	
	log.Printf("[RegisterLegacyRoutes] ✓ Legacy routes зарегистрированы")
}

