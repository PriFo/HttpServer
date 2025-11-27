package routes

import (
	"net/http"

	"httpserver/internal/api/handlers/upload"
	"httpserver/server/handlers"
)

// UploadHandlers содержит handlers для работы с upload
type UploadHandlers struct {
	// Новый handler из internal/api/handlers/upload
	Handler *upload.Handler
	// Legacy handler из server/handlers
	LegacyHandler *handlers.UploadLegacyHandler
	// Callback для /complete (например, для вызова qualityAnalyzer)
	CompleteCallback func(uploadID, databaseID int) error
	// Fallback handlers для старых методов Server
	HandleHandshake              http.HandlerFunc
	HandleMetadata               http.HandlerFunc
	HandleConstant               http.HandlerFunc
	HandleCatalogMeta            http.HandlerFunc
	HandleCatalogItem            http.HandlerFunc
	HandleCatalogItems           http.HandlerFunc
	HandleNomenclatureBatch      http.HandlerFunc
	HandleComplete               http.HandlerFunc
	HandleListUploads            http.HandlerFunc
	HandleUploadRoutes           http.HandlerFunc
	HandleNormalizedListUploads  http.HandlerFunc
	HandleNormalizedUploadRoutes http.HandlerFunc
	HandleNormalizedHandshake    http.HandlerFunc
	HandleNormalizedMetadata     http.HandlerFunc
	HandleNormalizedConstant     http.HandlerFunc
	HandleNormalizedCatalogMeta  http.HandlerFunc
	HandleNormalizedCatalogItem  http.HandlerFunc
	HandleNormalizedComplete     http.HandlerFunc
}

// RegisterUploadRoutes регистрирует маршруты для upload
func RegisterUploadRoutes(mux *http.ServeMux, h *UploadHandlers) {
	// Регистрируем /complete с оберткой для вызова qualityAnalyzer
	// Примечание: /complete регистрируется отдельно в server.go с оберткой для вызова qualityAnalyzer
	// Здесь мы регистрируем только если есть handler

	// Legacy endpoints для обратной совместимости с 1С
	if h.Handler != nil {
		mux.HandleFunc("/handshake", h.Handler.HandleHandshake)
		mux.HandleFunc("/metadata", h.Handler.HandleMetadata)
		mux.HandleFunc("/constant", h.Handler.HandleConstant)
		mux.HandleFunc("/catalog/meta", h.Handler.HandleCatalogMeta)
		mux.HandleFunc("/catalog/item", h.Handler.HandleCatalogItem)
		mux.HandleFunc("/catalog/items", h.Handler.HandleCatalogItems)
	} else if h.LegacyHandler != nil {
		mux.HandleFunc("/handshake", h.LegacyHandler.HandleHandshake)
		mux.HandleFunc("/metadata", h.LegacyHandler.HandleMetadata)
		mux.HandleFunc("/constant", h.LegacyHandler.HandleConstant)
		mux.HandleFunc("/catalog/meta", h.LegacyHandler.HandleCatalogMeta)
		mux.HandleFunc("/catalog/item", h.LegacyHandler.HandleCatalogItem)
		mux.HandleFunc("/catalog/items", h.LegacyHandler.HandleCatalogItems)
	} else {
		mux.HandleFunc("/handshake", h.HandleHandshake)
		mux.HandleFunc("/metadata", h.HandleMetadata)
		mux.HandleFunc("/constant", h.HandleConstant)
		mux.HandleFunc("/catalog/meta", h.HandleCatalogMeta)
		mux.HandleFunc("/catalog/item", h.HandleCatalogItem)
		mux.HandleFunc("/catalog/items", h.HandleCatalogItems)
	}

	// Регистрируем новые API v1 эндпоинты
	if h.Handler != nil {
		mux.HandleFunc("/api/v1/upload/handshake", h.Handler.HandleHandshake)
		mux.HandleFunc("/api/v1/upload/metadata", h.Handler.HandleMetadata)
		mux.HandleFunc("/api/v1/upload/nomenclature/batch", h.Handler.HandleNomenclatureBatch)
	} else if h.LegacyHandler != nil {
		mux.HandleFunc("/api/v1/upload/handshake", h.LegacyHandler.HandleHandshake)
		mux.HandleFunc("/api/v1/upload/metadata", h.LegacyHandler.HandleMetadata)
		mux.HandleFunc("/api/v1/upload/nomenclature/batch", h.LegacyHandler.HandleNomenclatureBatch)
	} else {
		mux.HandleFunc("/api/v1/upload/handshake", h.HandleHandshake)
		mux.HandleFunc("/api/v1/upload/metadata", h.HandleMetadata)
		mux.HandleFunc("/api/v1/upload/nomenclature/batch", h.HandleNomenclatureBatch)
	}

	// Регистрируем API эндпоинты
	// Новый handler имеет только HandleListUploads и HandleGetUpload
	// Остальные методы должны быть в fallback handlers
	if h.Handler != nil {
		mux.HandleFunc("/api/uploads", h.Handler.HandleListUploads)
		// HandleGetUpload обрабатывается через HandleUploadRoutes в server.go
		// или через fallback handlers ниже
	}

	// Регистрируем остальные API эндпоинты через fallback handlers
	// (новый handler не имеет этих методов)
	if h.HandleListUploads != nil {
		mux.HandleFunc("/api/uploads", h.HandleListUploads)
	}
	if h.HandleUploadRoutes != nil {
		mux.HandleFunc("/api/uploads/", h.HandleUploadRoutes)
	}

	// Регистрируем API эндпоинты для нормализованной БД
	if h.HandleNormalizedListUploads != nil {
		mux.HandleFunc("/api/normalized/uploads", h.HandleNormalizedListUploads)
	}
	if h.HandleNormalizedUploadRoutes != nil {
		mux.HandleFunc("/api/normalized/uploads/", h.HandleNormalizedUploadRoutes)
	}

	// Регистрируем эндпоинты для приема нормализованных данных
	if h.HandleNormalizedHandshake != nil {
		mux.HandleFunc("/api/normalized/upload/handshake", h.HandleNormalizedHandshake)
	}
	if h.HandleNormalizedMetadata != nil {
		mux.HandleFunc("/api/normalized/upload/metadata", h.HandleNormalizedMetadata)
	}
	if h.HandleNormalizedConstant != nil {
		mux.HandleFunc("/api/normalized/upload/constant", h.HandleNormalizedConstant)
	}
	if h.HandleNormalizedCatalogMeta != nil {
		mux.HandleFunc("/api/normalized/upload/catalog/meta", h.HandleNormalizedCatalogMeta)
	}
	if h.HandleNormalizedCatalogItem != nil {
		mux.HandleFunc("/api/normalized/upload/catalog/item", h.HandleNormalizedCatalogItem)
	}
	if h.HandleNormalizedComplete != nil {
		mux.HandleFunc("/api/normalized/upload/complete", h.HandleNormalizedComplete)
	}
}

// RegisterUploadRoutesWithMiddleware регистрирует маршруты с middleware
// Использует стандартную регистрацию, middleware применяется внутри handlers
func RegisterUploadRoutesWithMiddleware(
	mux *http.ServeMux,
	uploadHandler *upload.Handler,
	baseHandler *handlers.BaseHandler,
) {
	// Middleware применяется внутри каждого handler через baseHandler
	// Просто регистрируем маршруты напрямую
	RegisterUploadRoutes(mux, &UploadHandlers{
		Handler: uploadHandler,
	})
}

