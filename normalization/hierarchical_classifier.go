package normalization

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/nomenclature"
)

// ClassificationStep шаг классификации
type ClassificationStep struct {
	Level      KpvedLevel `json:"level"`
	LevelName  string     `json:"level_name"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Confidence float64    `json:"confidence"`
	Reasoning  string     `json:"reasoning"`
	Duration   int64      `json:"duration_ms"`
}

// HierarchicalResult результат иерархической классификации
type HierarchicalResult struct {
	FinalCode       string                `json:"final_code"`
	FinalName       string                `json:"final_name"`
	FinalConfidence float64               `json:"final_confidence"`
	Steps           []ClassificationStep  `json:"steps"`
	TotalDuration   int64                 `json:"total_duration_ms"`
	CacheHits       int                   `json:"cache_hits"`
	AICallsCount    int                   `json:"ai_calls_count"`
}

// AIResponse ответ от AI
type AIResponse struct {
	SelectedCode string  `json:"selected_code"`
	Confidence   float64 `json:"confidence"`
	Reasoning    string  `json:"reasoning"`
}

// HierarchicalClassifier иерархический классификатор КПВЭД
type HierarchicalClassifier struct {
	tree          *KpvedTree
	db            *database.DB
	aiClient      *nomenclature.AIClient
	promptBuilder *PromptBuilder
	cache         *sync.Map
	minConfidence float64 // минимальный порог уверенности для продолжения
}

// NewHierarchicalClassifier создает новый иерархический классификатор
func NewHierarchicalClassifier(db *database.DB, aiClient *nomenclature.AIClient) (*HierarchicalClassifier, error) {
	// Строим дерево из базы данных
	tree := NewKpvedTree()
	if err := tree.BuildFromDatabase(db); err != nil {
		return nil, fmt.Errorf("failed to build kpved tree: %w", err)
	}

	return &HierarchicalClassifier{
		tree:          tree,
		db:            db,
		aiClient:      aiClient,
		promptBuilder: NewPromptBuilder(tree),
		cache:         &sync.Map{},
		minConfidence: 0.7, // порог 70%
	}, nil
}

// Classify выполняет иерархическую классификацию
func (h *HierarchicalClassifier) Classify(normalizedName, category string) (*HierarchicalResult, error) {
	startTime := time.Now()
	result := &HierarchicalResult{
		Steps: make([]ClassificationStep, 0),
	}

	// Проверяем кэш для полной классификации
	cacheKey := h.getCacheKey(normalizedName, category, "")
	if cached, ok := h.cache.Load(cacheKey); ok {
		if cachedResult, ok := cached.(*HierarchicalResult); ok {
			log.Printf("[Cache] Hit for '%s' in '%s'", normalizedName, category)
			cachedResult.CacheHits++
			return cachedResult, nil
		}
	}

	// Шаг 1: Классификация по секциям (A-U)
	log.Printf("[Step 1/4] Classifying '%s' by section...", normalizedName)
	sectionStep, err := h.classifyLevel(normalizedName, category, LevelSection, "")
	if err != nil {
		return nil, fmt.Errorf("section classification failed: %w", err)
	}
	result.Steps = append(result.Steps, *sectionStep)
	result.AICallsCount++

	// Проверяем уверенность
	if sectionStep.Confidence < h.minConfidence {
		log.Printf("[Stop] Low confidence at section level: %.2f", sectionStep.Confidence)
		result.FinalCode = sectionStep.Code
		result.FinalName = sectionStep.Name
		result.FinalConfidence = sectionStep.Confidence
		result.TotalDuration = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Шаг 2: Классификация по классам (01, 02, ...)
	log.Printf("[Step 2/4] Classifying '%s' by class in section %s...", normalizedName, sectionStep.Code)
	classStep, err := h.classifyLevel(normalizedName, category, LevelClass, sectionStep.Code)
	if err != nil {
		return nil, fmt.Errorf("class classification failed: %w", err)
	}
	result.Steps = append(result.Steps, *classStep)
	result.AICallsCount++

	if classStep.Confidence < h.minConfidence {
		log.Printf("[Stop] Low confidence at class level: %.2f", classStep.Confidence)
		result.FinalCode = classStep.Code
		result.FinalName = classStep.Name
		result.FinalConfidence = classStep.Confidence
		result.TotalDuration = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Шаг 3: Классификация по подклассам (XX.Y)
	log.Printf("[Step 3/4] Classifying '%s' by subclass in class %s...", normalizedName, classStep.Code)
	subclassStep, err := h.classifyLevel(normalizedName, category, LevelSubclass, classStep.Code)
	if err != nil {
		return nil, fmt.Errorf("subclass classification failed: %w", err)
	}
	result.Steps = append(result.Steps, *subclassStep)
	result.AICallsCount++

	if subclassStep.Confidence < h.minConfidence {
		log.Printf("[Stop] Low confidence at subclass level: %.2f", subclassStep.Confidence)
		result.FinalCode = subclassStep.Code
		result.FinalName = subclassStep.Name
		result.FinalConfidence = subclassStep.Confidence
		result.TotalDuration = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Шаг 4: Классификация по группам (XX.YY)
	log.Printf("[Step 4/4] Classifying '%s' by group in subclass %s...", normalizedName, subclassStep.Code)
	groupStep, err := h.classifyLevel(normalizedName, category, LevelGroup, subclassStep.Code)
	if err != nil {
		return nil, fmt.Errorf("group classification failed: %w", err)
	}
	result.Steps = append(result.Steps, *groupStep)
	result.AICallsCount++

	// Финальный результат
	result.FinalCode = groupStep.Code
	result.FinalName = groupStep.Name
	result.FinalConfidence = groupStep.Confidence
	result.TotalDuration = time.Since(startTime).Milliseconds()

	// Сохраняем в кэш
	h.cache.Store(cacheKey, result)

	log.Printf("[Complete] Classified '%s' as %s (%s) with confidence %.2f in %dms",
		normalizedName, result.FinalCode, result.FinalName, result.FinalConfidence, result.TotalDuration)

	return result, nil
}

// classifyLevel классифицирует на указанном уровне
func (h *HierarchicalClassifier) classifyLevel(
	normalizedName, category string,
	level KpvedLevel,
	parentCode string,
) (*ClassificationStep, error) {
	stepStart := time.Now()

	// Проверяем кэш для уровня
	levelCacheKey := h.getCacheKey(normalizedName, category, string(level)+":"+parentCode)
	if cached, ok := h.cache.Load(levelCacheKey); ok {
		if cachedStep, ok := cached.(*ClassificationStep); ok {
			log.Printf("[Cache] Hit for level %s with parent %s", level, parentCode)
			return cachedStep, nil
		}
	}

	// Получаем кандидатов для этого уровня
	candidates := h.tree.GetNodesAtLevel(level, parentCode)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates found for level %s with parent %s", level, parentCode)
	}

	log.Printf("[Level %s] Found %d candidates", level, len(candidates))

	// Строим промпт
	prompt := h.promptBuilder.BuildLevelPrompt(normalizedName, category, level, candidates)

	// Вызываем AI
	systemPrompt := prompt.System
	userPrompt := prompt.User

	log.Printf("[AI Call] Level: %s, Prompt size: %d bytes", level, prompt.GetPromptSize())

	response, err := h.aiClient.GetCompletion(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("ai call failed: %w", err)
	}

	// Парсим ответ
	aiResponse, err := h.parseAIResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ai response: %w", err)
	}

	// Находим выбранный узел
	selectedNode, exists := h.tree.GetNode(aiResponse.SelectedCode)
	if !exists {
		return nil, fmt.Errorf("selected code %s not found in tree", aiResponse.SelectedCode)
	}

	// Создаем шаг
	step := &ClassificationStep{
		Level:      level,
		LevelName:  GetLevelName(level),
		Code:       selectedNode.Code,
		Name:       selectedNode.Name,
		Confidence: aiResponse.Confidence,
		Reasoning:  aiResponse.Reasoning,
		Duration:   time.Since(stepStart).Milliseconds(),
	}

	// Сохраняем в кэш
	h.cache.Store(levelCacheKey, step)

	return step, nil
}

// parseAIResponse парсит JSON ответ от AI
func (h *HierarchicalClassifier) parseAIResponse(response string) (*AIResponse, error) {
	// Убираем markdown-обертки, если есть
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var aiResp AIResponse
	if err := json.Unmarshal([]byte(response), &aiResp); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w, response: %s", err, response)
	}

	// Валидация
	if aiResp.SelectedCode == "" {
		return nil, fmt.Errorf("empty selected_code in response")
	}

	if aiResp.Confidence < 0 || aiResp.Confidence > 1 {
		// Если уверенность в процентах (0-100), конвертируем
		if aiResp.Confidence > 1 && aiResp.Confidence <= 100 {
			aiResp.Confidence = aiResp.Confidence / 100.0
		} else {
			return nil, fmt.Errorf("invalid confidence value: %f", aiResp.Confidence)
		}
	}

	return &aiResp, nil
}

// getCacheKey генерирует ключ кэша
func (h *HierarchicalClassifier) getCacheKey(normalizedName, category, suffix string) string {
	if suffix != "" {
		return fmt.Sprintf("%s:%s:%s", normalizedName, category, suffix)
	}
	return fmt.Sprintf("%s:%s", normalizedName, category)
}

// ClearCache очищает кэш
func (h *HierarchicalClassifier) ClearCache() {
	h.cache = &sync.Map{}
	log.Println("[Cache] Cleared")
}

// GetCacheStats возвращает статистику кэша
func (h *HierarchicalClassifier) GetCacheStats() map[string]int {
	count := 0
	h.cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return map[string]int{
		"entries": count,
	}
}

// SetMinConfidence устанавливает минимальный порог уверенности
func (h *HierarchicalClassifier) SetMinConfidence(confidence float64) {
	if confidence >= 0 && confidence <= 1 {
		h.minConfidence = confidence
		log.Printf("[Config] Min confidence set to %.2f", confidence)
	}
}
