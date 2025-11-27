package algorithms

import (
	"strings"
	"sync"
)

// PrefixIndex префиксный индекс для быстрой фильтрации кандидатов
// Используется для оптимизации поиска дубликатов - позволяет быстро отсеивать
// заведомо разные строки перед применением дорогих алгоритмов сравнения
type PrefixIndex struct {
	// Индекс: префикс -> список индексов элементов
	index map[string][]int
	
	// Обратный индекс: индекс элемента -> список префиксов
	reverseIndex map[int][]string
	
	// Длина префикса
	prefixLength int
	
	// Минимальная длина строки для индексации
	minLength int
	
	// Защита от конкурентного доступа
	mu sync.RWMutex
}

// NewPrefixIndex создает новый префиксный индекс
// prefixLength - длина префикса для индексации (рекомендуется 3-5 символов)
// minLength - минимальная длина строки для индексации
func NewPrefixIndex(prefixLength, minLength int) *PrefixIndex {
	if prefixLength <= 0 {
		prefixLength = 3 // По умолчанию 3 символа
	}
	if minLength <= 0 {
		minLength = prefixLength
	}
	
	return &PrefixIndex{
		index:        make(map[string][]int),
		reverseIndex: make(map[int][]string),
		prefixLength: prefixLength,
		minLength:    minLength,
	}
}

// Add добавляет строку в индекс
func (pi *PrefixIndex) Add(index int, text string) {
	if text == "" || len(text) < pi.minLength {
		return
	}
	
	normalized := strings.ToLower(strings.TrimSpace(text))
	if len(normalized) < pi.minLength {
		return
	}
	
	// Извлекаем префикс
	prefix := pi.getPrefix(normalized)
	if prefix == "" {
		return
	}
	
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	// Добавляем в индекс
	pi.index[prefix] = append(pi.index[prefix], index)
	
	// Добавляем в обратный индекс
	if pi.reverseIndex[index] == nil {
		pi.reverseIndex[index] = make([]string, 0)
	}
	pi.reverseIndex[index] = append(pi.reverseIndex[index], prefix)
}

// AddBatch добавляет несколько строк в индекс
func (pi *PrefixIndex) AddBatch(texts []string) {
	for i, text := range texts {
		pi.Add(i, text)
	}
}

// GetCandidates возвращает список индексов кандидатов для сравнения
// Использует префиксную фильтрацию для быстрого отсеивания
func (pi *PrefixIndex) GetCandidates(index int, text string) []int {
	if text == "" || len(text) < pi.minLength {
		return []int{}
	}
	
	normalized := strings.ToLower(strings.TrimSpace(text))
	if len(normalized) < pi.minLength {
		return []int{}
	}
	
	prefix := pi.getPrefix(normalized)
	if prefix == "" {
		return []int{}
	}
	
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	// Получаем кандидатов с тем же префиксом
	candidates := make(map[int]bool)
	if indices, exists := pi.index[prefix]; exists {
		for _, idx := range indices {
			if idx != index {
				candidates[idx] = true
			}
		}
	}
	
	// Также добавляем кандидатов с похожими префиксами (для учета опечаток)
	// Проверяем префиксы, отличающиеся на 1-2 символа
	for p, indices := range pi.index {
		if p != prefix && pi.prefixSimilarity(prefix, p) >= 0.5 {
			for _, idx := range indices {
				if idx != index {
					candidates[idx] = true
				}
			}
		}
	}
	
	// Преобразуем map в slice
	result := make([]int, 0, len(candidates))
	for idx := range candidates {
		result = append(result, idx)
	}
	
	return result
}

// GetCandidatesExact возвращает только кандидатов с точно таким же префиксом
// Более строгая фильтрация, быстрее работает
func (pi *PrefixIndex) GetCandidatesExact(index int, text string) []int {
	if text == "" || len(text) < pi.minLength {
		return []int{}
	}
	
	normalized := strings.ToLower(strings.TrimSpace(text))
	if len(normalized) < pi.minLength {
		return []int{}
	}
	
	prefix := pi.getPrefix(normalized)
	if prefix == "" {
		return []int{}
	}
	
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	if indices, exists := pi.index[prefix]; exists {
		result := make([]int, 0, len(indices))
		for _, idx := range indices {
			if idx != index {
				result = append(result, idx)
			}
		}
		return result
	}
	
	return []int{}
}

// getPrefix извлекает префикс из строки
func (pi *PrefixIndex) getPrefix(text string) string {
	if len(text) < pi.prefixLength {
		return text
	}
	return text[:pi.prefixLength]
}

// prefixSimilarity вычисляет схожесть двух префиксов
// Возвращает значение от 0.0 до 1.0
func (pi *PrefixIndex) prefixSimilarity(p1, p2 string) float64 {
	if len(p1) == 0 && len(p2) == 0 {
		return 1.0
	}
	if len(p1) == 0 || len(p2) == 0 {
		return 0.0
	}
	
	// Преобразуем в руны для правильного сравнения Unicode символов
	runes1 := []rune(p1)
	runes2 := []rune(p2)
	
	// Используем минимальную длину для сравнения
	minLen := len(runes1)
	if len(runes2) < minLen {
		minLen = len(runes2)
	}
	
	if minLen == 0 {
		return 0.0
	}
	
	matches := 0
	for i := 0; i < minLen; i++ {
		if runes1[i] == runes2[i] {
			matches++
		}
	}
	
	return float64(matches) / float64(minLen)
}

// Clear очищает индекс
func (pi *PrefixIndex) Clear() {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	pi.index = make(map[string][]int)
	pi.reverseIndex = make(map[int][]string)
}

// GetStats возвращает статистику индекса
func (pi *PrefixIndex) GetStats() PrefixIndexStats {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	totalItems := len(pi.reverseIndex)
	totalPrefixes := len(pi.index)
	
	// Вычисляем среднее количество элементов на префикс
	avgItemsPerPrefix := 0.0
	if totalPrefixes > 0 {
		totalItemsInPrefixes := 0
		for _, indices := range pi.index {
			totalItemsInPrefixes += len(indices)
		}
		avgItemsPerPrefix = float64(totalItemsInPrefixes) / float64(totalPrefixes)
	}
	
	return PrefixIndexStats{
		TotalItems:         totalItems,
		TotalPrefixes:      totalPrefixes,
		AvgItemsPerPrefix: avgItemsPerPrefix,
		PrefixLength:      pi.prefixLength,
		MinLength:         pi.minLength,
	}
}

// PrefixIndexStats статистика префиксного индекса
type PrefixIndexStats struct {
	TotalItems         int     `json:"total_items"`
	TotalPrefixes      int     `json:"total_prefixes"`
	AvgItemsPerPrefix  float64 `json:"avg_items_per_prefix"`
	PrefixLength       int     `json:"prefix_length"`
	MinLength          int     `json:"min_length"`
}

// Remove удаляет элемент из индекса
func (pi *PrefixIndex) Remove(index int) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	// Получаем префиксы для этого элемента
	prefixes, exists := pi.reverseIndex[index]
	if !exists {
		return
	}
	
	// Удаляем из основного индекса
	for _, prefix := range prefixes {
		if indices, exists := pi.index[prefix]; exists {
			newIndices := make([]int, 0, len(indices))
			for _, idx := range indices {
				if idx != index {
					newIndices = append(newIndices, idx)
				}
			}
			if len(newIndices) == 0 {
				delete(pi.index, prefix)
			} else {
				pi.index[prefix] = newIndices
			}
		}
	}
	
	// Удаляем из обратного индекса
	delete(pi.reverseIndex, index)
}

// Update обновляет элемент в индексе
func (pi *PrefixIndex) Update(index int, oldText, newText string) {
	pi.Remove(index)
	pi.Add(index, newText)
}

// GetPrefixes возвращает все префиксы для элемента
func (pi *PrefixIndex) GetPrefixes(index int) []string {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	if prefixes, exists := pi.reverseIndex[index]; exists {
		result := make([]string, len(prefixes))
		copy(result, prefixes)
		return result
	}
	
	return []string{}
}

// FilterByPrefix фильтрует список индексов по префиксу
func (pi *PrefixIndex) FilterByPrefix(prefix string, indices []int) []int {
	normalizedPrefix := strings.ToLower(strings.TrimSpace(prefix))
	if len(normalizedPrefix) < pi.prefixLength {
		normalizedPrefix = pi.getPrefix(normalizedPrefix)
	} else {
		normalizedPrefix = normalizedPrefix[:pi.prefixLength]
	}
	
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	if prefixIndices, exists := pi.index[normalizedPrefix]; exists {
		// Создаем map для быстрого поиска
		prefixMap := make(map[int]bool)
		for _, idx := range prefixIndices {
			prefixMap[idx] = true
		}
		
		// Фильтруем исходный список
		result := make([]int, 0)
		for _, idx := range indices {
			if prefixMap[idx] {
				result = append(result, idx)
			}
		}
		
		return result
	}
	
	return []int{}
}

