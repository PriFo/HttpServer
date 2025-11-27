package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SystemSummaryFilter параметры фильтрации для SystemSummary
type SystemSummaryFilter struct {
	Status        []string   // Фильтр по статусу (completed, failed, in_progress)
	CreatedAfter  *time.Time // Фильтр по дате создания (после)
	CreatedBefore *time.Time // Фильтр по дате создания (до)
	Search        string     // Поиск по имени загрузки
	Limit         int        // Лимит для upload_details (0 = все)
	Offset        int        // Смещение для upload_details
	SortBy        string     // Поле для сортировки (created_at, completed_at, name, status, nomenclature_count, counterparty_count)
	Order         string     // Направление сортировки (asc, desc)
}

// ApplyFilters применяет фильтры к SystemSummary
func ApplyFilters(summary *SystemSummary, filter SystemSummaryFilter) *SystemSummary {
	if summary == nil {
		return summary
	}

	// Создаем копию для изменения
	filtered := &SystemSummary{
		TotalDatabases:      summary.TotalDatabases,
		TotalUploads:        summary.TotalUploads,
		CompletedUploads:    summary.CompletedUploads,
		FailedUploads:       summary.FailedUploads,
		InProgressUploads:   summary.InProgressUploads,
		LastActivity:        summary.LastActivity,
		TotalNomenclature:   summary.TotalNomenclature,
		TotalCounterparties: summary.TotalCounterparties,
		UploadDetails:       make([]UploadSummary, 0),
		ScanDuration:        summary.ScanDuration,
		DatabasesProcessed:  summary.DatabasesProcessed,
		DatabasesSkipped:    summary.DatabasesSkipped,
	}

	// Фильтруем upload_details
	statusMap := make(map[string]bool)
	for _, status := range filter.Status {
		if status != "" {
			statusMap[strings.ToLower(status)] = true
		}
	}

	searchLower := strings.ToLower(filter.Search)
	filteredDetails := make([]UploadSummary, 0)

	for _, upload := range summary.UploadDetails {
		// Фильтр по статусу
		if len(statusMap) > 0 {
			if !statusMap[strings.ToLower(upload.Status)] {
				continue
			}
		}

		// Фильтр по дате создания (после)
		if filter.CreatedAfter != nil {
			if upload.CreatedAt.Before(*filter.CreatedAfter) {
				continue
			}
		}

		// Фильтр по дате создания (до)
		if filter.CreatedBefore != nil {
			if upload.CreatedAt.After(*filter.CreatedBefore) {
				continue
			}
		}

		// Поиск по имени
		if searchLower != "" {
			nameLower := strings.ToLower(upload.Name)
			uuidLower := strings.ToLower(upload.UploadUUID)
			if !strings.Contains(nameLower, searchLower) && !strings.Contains(uuidLower, searchLower) {
				continue
			}
		}

		filteredDetails = append(filteredDetails, upload)
	}

	// Сортировка
	if filter.SortBy != "" {
		sort.Slice(filteredDetails, func(i, j int) bool {
			return compareUploadSummary(filteredDetails[i], filteredDetails[j], filter.SortBy, filter.Order == "desc")
		})
	}

	// Пагинация
	total := len(filteredDetails)
	if filter.Limit > 0 {
		start := filter.Offset
		end := filter.Offset + filter.Limit
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		if start < 0 {
			start = 0
		}
		if start < end {
			filteredDetails = filteredDetails[start:end]
		} else {
			filteredDetails = []UploadSummary{}
		}
	}

	filtered.UploadDetails = filteredDetails

	// Пересчитываем статистику по отфильтрованным данным
	filtered.CompletedUploads = 0
	filtered.FailedUploads = 0
	filtered.InProgressUploads = 0
	filtered.TotalNomenclature = 0
	filtered.TotalCounterparties = 0

	for _, upload := range filteredDetails {
		switch upload.Status {
		case "completed":
			filtered.CompletedUploads++
		case "failed":
			filtered.FailedUploads++
		case "in_progress":
			filtered.InProgressUploads++
		}
		filtered.TotalNomenclature += upload.NomenclatureCount
		filtered.TotalCounterparties += upload.CounterpartyCount
	}

	filtered.TotalUploads = int64(len(filteredDetails))

	return filtered
}

// compareUploadSummary сравнивает два UploadSummary для сортировки
func compareUploadSummary(a, b UploadSummary, sortBy string, descending bool) bool {
	var result bool

	switch sortBy {
	case "created_at":
		result = a.CreatedAt.Before(b.CreatedAt)
	case "completed_at":
		if a.CompletedAt == nil && b.CompletedAt == nil {
			result = false
		} else if a.CompletedAt == nil {
			result = true
		} else if b.CompletedAt == nil {
			result = false
		} else {
			result = a.CompletedAt.Before(*b.CompletedAt)
		}
	case "name":
		result = strings.ToLower(a.Name) < strings.ToLower(b.Name)
	case "status":
		result = a.Status < b.Status
	case "nomenclature_count":
		result = a.NomenclatureCount < b.NomenclatureCount
	case "counterparty_count":
		result = a.CounterpartyCount < b.CounterpartyCount
	default:
		result = a.CreatedAt.Before(b.CreatedAt) // По умолчанию сортируем по дате создания
	}

	if descending {
		return !result
	}
	return result
}

// ParseSystemSummaryFilterFromRequest парсит query параметры из http.Request в SystemSummaryFilter
func ParseSystemSummaryFilterFromRequest(r *http.Request) SystemSummaryFilter {
	filter := SystemSummaryFilter{}
	query := r.URL.Query()

	// Вспомогательная функция для получения первого значения параметра
	getParam := func(key string) string {
		if values := query[key]; len(values) > 0 {
			return values[0]
		}
		return ""
	}

	// Фильтр по статусу (может быть несколько: ?status=completed,failed или ?status=completed&status=failed)
	if statusParam := getParam("status"); statusParam != "" {
		statusList := strings.Split(statusParam, ",")
		for _, status := range statusList {
			status = strings.TrimSpace(strings.ToLower(status))
			if status != "" {
				filter.Status = append(filter.Status, status)
			}
		}
	}
	// Также обрабатываем множественные параметры status=completed&status=failed
	if statuses := query["status"]; len(statuses) > 1 {
		for _, status := range statuses {
			status = strings.TrimSpace(strings.ToLower(status))
			if status != "" {
				// Проверяем, что еще не добавлен
				found := false
				for _, existing := range filter.Status {
					if existing == status {
						found = true
						break
					}
				}
				if !found {
					filter.Status = append(filter.Status, status)
				}
			}
		}
	}

	// Поиск
	if search := getParam("search"); search != "" {
		filter.Search = strings.TrimSpace(search)
	}

	// Фильтр по дате создания (после)
	if createdAfterStr := getParam("created_after"); createdAfterStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAfterStr); err == nil {
			filter.CreatedAfter = &t
		}
	}

	// Фильтр по дате создания (до)
	if createdBeforeStr := getParam("created_before"); createdBeforeStr != "" {
		if t, err := time.Parse(time.RFC3339, createdBeforeStr); err == nil {
			filter.CreatedBefore = &t
		}
	}

	// Сортировка
	filter.SortBy = strings.ToLower(strings.TrimSpace(getParam("sort_by")))
	if filter.SortBy != "" {
		validSortFields := map[string]bool{
			"created_at":         true,
			"completed_at":       true,
			"name":               true,
			"status":             true,
			"nomenclature_count": true,
			"counterparty_count": true,
		}
		if !validSortFields[filter.SortBy] {
			filter.SortBy = "created_at" // По умолчанию
		}
	} else {
		filter.SortBy = "created_at" // По умолчанию
	}

	filter.Order = strings.ToLower(strings.TrimSpace(getParam("order")))
	if filter.Order != "asc" && filter.Order != "desc" {
		filter.Order = "desc" // По умолчанию новые первыми
	}

	// Пагинация
	if limitStr := getParam("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	if pageStr := getParam("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page >= 1 {
			if filter.Limit > 0 {
				filter.Offset = (page - 1) * filter.Limit
			}
		}
	}

	return filter
}
