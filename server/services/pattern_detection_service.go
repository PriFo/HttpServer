package services

import (
	"os"

	"httpserver/normalization"
	apperrors "httpserver/server/errors"
)

// PatternDetectionService сервис для обнаружения паттернов
type PatternDetectionService struct {
	getArliaiAPIKey func() string
}

// NewPatternDetectionService создает новый сервис для обнаружения паттернов
func NewPatternDetectionService(getArliaiAPIKey func() string) *PatternDetectionService {
	return &PatternDetectionService{
		getArliaiAPIKey: getArliaiAPIKey,
	}
}

// DetectPatterns обнаруживает паттерны в названии
func (s *PatternDetectionService) DetectPatterns(name string) (map[string]interface{}, error) {
	if name == "" {
		return nil, apperrors.NewValidationError("имя обязательно", nil)
	}

	detector := normalization.NewPatternDetector()
	matches := detector.DetectPatterns(name)
	algorithmicFix := detector.ApplyFixes(name, matches)
	summary := detector.GetPatternSummary(matches)

	return map[string]interface{}{
		"original_name":     name,
		"patterns":          matches,
		"algorithmic_fix":   algorithmicFix,
		"summary":           summary,
		"patterns_count":    len(matches),
	}, nil
}

// SuggestPatternCorrection предлагает исправление с использованием паттернов и AI
func (s *PatternDetectionService) SuggestPatternCorrection(name string, useAI bool) (map[string]interface{}, error) {
	if name == "" {
		return nil, apperrors.NewValidationError("имя обязательно", nil)
	}

	detector := normalization.NewPatternDetector()
	matches := detector.DetectPatterns(name)
	algorithmicFix := detector.ApplyFixes(name, matches)

	result := map[string]interface{}{
		"original_name":    name,
		"patterns":         matches,
		"algorithmic_fix":  algorithmicFix,
		"patterns_count":   len(matches),
	}

	if useAI {
		apiKey := s.getArliaiAPIKey()
		if apiKey == "" {
			apiKey = os.Getenv("ARLIAI_API_KEY")
		}
		if apiKey != "" {
			aiNormalizer := normalization.NewAINormalizer(apiKey)
			aiIntegrator := normalization.NewPatternAIIntegrator(detector, aiNormalizer)

			aiResult, err := aiIntegrator.SuggestCorrectionWithAI(name)
			if err == nil {
				result["ai_suggested_fix"] = aiResult.AISuggestedFix
				result["final_suggestion"] = aiResult.FinalSuggestion
				result["confidence"] = aiResult.Confidence
				result["reasoning"] = aiResult.Reasoning
				result["requires_review"] = aiResult.RequiresReview
			} else {
				result["ai_error"] = err.Error()
				result["final_suggestion"] = algorithmicFix
			}
		} else {
			result["ai_error"] = "ARLIAI_API_KEY not set"
			result["final_suggestion"] = algorithmicFix
		}
	} else {
		result["final_suggestion"] = algorithmicFix
	}

	return result, nil
}

// TestPatternsBatch тестирует паттерны на выборке данных
func (s *PatternDetectionService) TestPatternsBatch(limit int, useAI bool, table, column string, getNames func(limit int, table, column string) ([]string, error)) (map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if table == "" {
		table = "catalog_items"
	}
	if column == "" {
		column = "name"
	}

	detector := normalization.NewPatternDetector()

	var aiIntegrator *normalization.PatternAIIntegrator
	if useAI {
		apiKey := s.getArliaiAPIKey()
		if apiKey == "" {
			apiKey = os.Getenv("ARLIAI_API_KEY")
		}
		if apiKey != "" {
			aiNormalizer := normalization.NewAINormalizer(apiKey)
			aiIntegrator = normalization.NewPatternAIIntegrator(detector, aiNormalizer)
		}
	}

	names, err := getNames(limit, table, column)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить имена", err)
	}

	results := make([]map[string]interface{}, 0, len(names))
	for _, name := range names {
		matches := detector.DetectPatterns(name)
		result := map[string]interface{}{
			"original_name":    name,
			"patterns_found":   len(matches),
			"patterns":         matches,
		}

		algorithmicFix := detector.ApplyFixes(name, matches)
		result["algorithmic_fix"] = algorithmicFix

		if aiIntegrator != nil {
			aiResult, err := aiIntegrator.SuggestCorrectionWithAI(name)
			if err == nil {
				result["ai_suggested_fix"] = aiResult.AISuggestedFix
				result["final_suggestion"] = aiResult.FinalSuggestion
				result["confidence"] = aiResult.Confidence
				result["reasoning"] = aiResult.Reasoning
				result["requires_review"] = aiResult.RequiresReview
			} else {
				result["ai_error"] = err.Error()
				result["final_suggestion"] = algorithmicFix
			}
		} else {
			result["final_suggestion"] = algorithmicFix
		}

		results = append(results, result)
	}

	stats := s.calculatePatternStats(results)

	return map[string]interface{}{
		"total_analyzed": len(results),
		"results":        results,
		"statistics":     stats,
	}, nil
}

// calculatePatternStats вычисляет статистику по результатам анализа паттернов
func (s *PatternDetectionService) calculatePatternStats(results []map[string]interface{}) map[string]interface{} {
	stats := make(map[string]interface{})

	totalPatterns := 0
	patternTypes := make(map[string]int)
	severityCount := make(map[string]int)
	autoFixableCount := 0
	itemsWithPatterns := 0
	itemsRequiringReview := 0

	for _, result := range results {
		patternsCount := 0
		if count, ok := result["patterns_found"].(int); ok {
			patternsCount = count
		}

		if patternsCount > 0 {
			itemsWithPatterns++
		}

		if patterns, ok := result["patterns"].([]normalization.PatternMatch); ok {
			totalPatterns += len(patterns)
			for _, match := range patterns {
				patternTypes[string(match.Type)]++
				severityCount[match.Severity]++
				if match.AutoFixable {
					autoFixableCount++
				}
			}
		}

		if requiresReview, ok := result["requires_review"].(bool); ok && requiresReview {
			itemsRequiringReview++
		}
	}

	stats["total_patterns"] = totalPatterns
	stats["items_with_patterns"] = itemsWithPatterns
	stats["items_requiring_review"] = itemsRequiringReview
	stats["auto_fixable_patterns"] = autoFixableCount
	stats["patterns_by_type"] = patternTypes
	stats["patterns_by_severity"] = severityCount

	if len(results) > 0 {
		stats["avg_patterns_per_item"] = float64(totalPatterns) / float64(len(results))
		stats["items_with_patterns_percent"] = float64(itemsWithPatterns) / float64(len(results)) * 100
	}

	return stats
}

