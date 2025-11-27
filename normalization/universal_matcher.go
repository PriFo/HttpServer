package normalization

import (
	"sync"

	"httpserver/normalization/algorithms"
)

// UniversalMatcher универсальный матчер с единым интерфейсом для всех методов
type UniversalMatcher struct {
	methods  map[string]algorithms.SimilarityMethod
	cache    map[string]float64
	cacheMu  sync.RWMutex
	useCache bool
}

// NewUniversalMatcher создает новый универсальный матчер
func NewUniversalMatcher(useCache bool) *UniversalMatcher {
	um := &UniversalMatcher{
		methods:  make(map[string]algorithms.SimilarityMethod),
		cache:    make(map[string]float64),
		useCache: useCache,
	}

	// Регистрируем методы по умолчанию
	um.RegisterDefaultMethods()

	return um
}

// RegisterMethod регистрирует новый метод
func (um *UniversalMatcher) RegisterMethod(name string, method algorithms.SimilarityMethod) {
	um.methods[name] = method
}

// RegisterDefaultMethods регистрирует все доступные методы по умолчанию
func (um *UniversalMatcher) RegisterDefaultMethods() {
	defaultMethods := algorithms.GetDefaultMethods()
	for _, method := range defaultMethods {
		um.methods[method.Name] = method
	}

	// Дополнительные методы
	// Дополнительные методы будут добавлены через RegisterMethod
	// или могут быть добавлены позже при необходимости
}

// Similarity вычисляет сходство используя указанный метод
func (um *UniversalMatcher) Similarity(s1, s2 string, methodName string) (float64, error) {
	method, exists := um.methods[methodName]
	if !exists {
		return 0.0, &MethodNotFoundError{Method: methodName}
	}

	// Проверяем кэш
	if um.useCache {
		cacheKey := s1 + "|" + s2 + "|" + methodName
		um.cacheMu.RLock()
		if cached, ok := um.cache[cacheKey]; ok {
			um.cacheMu.RUnlock()
			return cached, nil
		}
		um.cacheMu.RUnlock()
	}

	// Вычисляем сходство
	similarity := method.Compute(s1, s2)

	// Сохраняем в кэш
	if um.useCache {
		cacheKey := s1 + "|" + s2 + "|" + methodName
		um.cacheMu.Lock()
		um.cache[cacheKey] = similarity
		um.cacheMu.Unlock()
	}

	return similarity, nil
}

// SimilarityMultiple вычисляет сходство используя несколько методов
func (um *UniversalMatcher) SimilarityMultiple(s1, s2 string, methodNames []string) (map[string]float64, error) {
	results := make(map[string]float64)

	for _, methodName := range methodNames {
		similarity, err := um.Similarity(s1, s2, methodName)
		if err != nil {
			return nil, err
		}
		results[methodName] = similarity
	}

	return results, nil
}

// IsMatch проверяет, являются ли строки совпадением используя указанный метод
func (um *UniversalMatcher) IsMatch(s1, s2 string, methodName string) (bool, error) {
	method, exists := um.methods[methodName]
	if !exists {
		return false, &MethodNotFoundError{Method: methodName}
	}

	similarity := method.Compute(s1, s2)
	return similarity >= method.Threshold, nil
}

// HybridSimilarity вычисляет гибридное сходство используя несколько методов
func (um *UniversalMatcher) HybridSimilarity(s1, s2 string, methodNames []string, weights []float64) (float64, error) {
	if len(methodNames) == 0 {
		return 0.0, &NoMethodsError{}
	}

	methods := make([]algorithms.SimilarityMethod, 0, len(methodNames))
	for _, name := range methodNames {
		method, exists := um.methods[name]
		if !exists {
			return 0.0, &MethodNotFoundError{Method: name}
		}
		methods = append(methods, method)
	}

	similarity := algorithms.WeightedSimilarity(s1, s2, methods, weights)
	return similarity, nil
}

// HybridSimilarityAdvanced вычисляет гибридное сходство используя улучшенный алгоритм
// Использует HybridSimilarityAdvanced из algorithms пакета с оптимизированными весами
func (um *UniversalMatcher) HybridSimilarityAdvanced(s1, s2 string, weights *algorithms.SimilarityWeights) (float64, error) {
	// Проверяем кэш
	if um.useCache {
		cacheKey := s1 + "|" + s2 + "|advanced"
		um.cacheMu.RLock()
		if cached, ok := um.cache[cacheKey]; ok {
			um.cacheMu.RUnlock()
			return cached, nil
		}
		um.cacheMu.RUnlock()
	}

	// Используем улучшенный гибридный алгоритм
	similarity := algorithms.HybridSimilarityAdvanced(s1, s2, weights)

	// Сохраняем в кэш
	if um.useCache {
		cacheKey := s1 + "|" + s2 + "|advanced"
		um.cacheMu.Lock()
		um.cache[cacheKey] = similarity
		um.cacheMu.Unlock()
	}

	return similarity, nil
}

// EnsembleSimilarity вычисляет ансамблевое сходство
func (um *UniversalMatcher) EnsembleSimilarity(s1, s2 string, methodNames []string, strategy algorithms.VotingStrategy) (float64, error) {
	if len(methodNames) == 0 {
		return 0.0, &NoMethodsError{}
	}

	methods := make([]algorithms.SimilarityMethod, 0, len(methodNames))
	for _, name := range methodNames {
		method, exists := um.methods[name]
		if !exists {
			return 0.0, &MethodNotFoundError{Method: name}
		}
		methods = append(methods, method)
	}

	threshold := 0.85 // Порог по умолчанию
	ensemble := algorithms.NewEnsembleMatcher(methods, threshold, strategy)
	similarity := ensemble.Similarity(s1, s2)

	return similarity, nil
}

// GetAvailableMethods возвращает список доступных методов
func (um *UniversalMatcher) GetAvailableMethods() []string {
	methods := make([]string, 0, len(um.methods))
	for name := range um.methods {
		methods = append(methods, name)
	}
	return methods
}

// ClearCache очищает кэш
func (um *UniversalMatcher) ClearCache() {
	um.cacheMu.Lock()
	um.cache = make(map[string]float64)
	um.cacheMu.Unlock()
}

// MethodNotFoundError ошибка когда метод не найден
type MethodNotFoundError struct {
	Method string
}

func (e *MethodNotFoundError) Error() string {
	return "method not found: " + e.Method
}

// NoMethodsError ошибка когда не указаны методы
type NoMethodsError struct{}

func (e *NoMethodsError) Error() string {
	return "no methods specified"
}
