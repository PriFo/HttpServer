package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apperrors "httpserver/server/errors"
)

// HandleClassifyItemGin обработчик классификации элемента для Gin
// @Summary Классифицировать элемент
// @Description Классифицирует элемент номенклатуры по справочнику КПВЭД
// @Tags classification
// @Accept json
// @Produce json
// @Param request body ClassifyRequest true "Данные для классификации"
// @Success 200 {object} ClassifyResponse "Результат классификации"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/classification/classify [post]
func (h *ClassificationHandler) HandleClassifyItemGin(c *gin.Context) {
	var req ClassifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := apperrors.NewValidationError("неверный формат тела запроса", err)
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	if req.ItemID == 0 && req.ItemName == "" {
		SendJSONError(c, http.StatusBadRequest, "item_id or item_name is required")
		return
	}

	// Здесь должна быть логика классификации
	response := ClassifyResponse{
		ItemID:     req.ItemID,
		ItemName:   req.ItemName,
		Classifier: "kpved",
		Confidence: 0.0,
	}

	SendJSONResponse(c, http.StatusOK, response)
}

// HandleClassificationStatsGin обработчик статистики классификации для Gin
// @Summary Получить статистику классификации
// @Description Возвращает статистику процесса классификации
// @Tags classification
// @Accept json
// @Produce json
// @Param database_id query int false "ID базы данных"
// @Success 200 {object} ClassificationStatsResponse "Статистика классификации"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/classification/stats [get]
func (h *ClassificationHandler) HandleClassificationStatsGin(c *gin.Context) {
	databaseIDStr := c.Query("database_id")
	if databaseIDStr != "" {
		_, err := strconv.Atoi(databaseIDStr)
		if err != nil {
			appErr := apperrors.NewValidationError("неверный формат database_id", err)
			SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
			return
		}
		// databaseID может быть использован для фильтрации статистики
	}

	// Здесь должна быть логика получения статистики
	response := ClassificationStatsResponse{
		TotalItems:        0,
		ClassifiedItems:   0,
		PendingItems:      0,
		AverageConfidence: 0.0,
	}

	SendJSONResponse(c, http.StatusOK, response)
}
