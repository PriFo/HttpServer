package algorithms

import (
	"testing"
)

// BenchmarkLevenshteinSimilarity бенчмарк для Levenshtein
func BenchmarkLevenshteinSimilarity(b *testing.B) {
	s1 := "кабель медный ВВГ 3x2.5"
	s2 := "кабель медный ВВГ 3x2,5"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LevenshteinSimilarity(s1, s2)
	}
}

// BenchmarkDamerauLevenshteinSimilarity бенчмарк для Damerau-Levenshtein
func BenchmarkDamerauLevenshteinSimilarity(b *testing.B) {
	s1 := "кабель медный ВВГ 3x2.5"
	s2 := "кабель медный ВВГ 3x2,5"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DamerauLevenshteinSimilarity(s1, s2)
	}
}

// BenchmarkJaroSimilarity бенчмарк для Jaro
func BenchmarkJaroSimilarity(b *testing.B) {
	s1 := "кабель медный ВВГ 3x2.5"
	s2 := "кабель медный ВВГ 3x2,5"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JaroSimilarity(s1, s2)
	}
}

// BenchmarkJaroWinklerSimilarity бенчмарк для Jaro-Winkler
func BenchmarkJaroWinklerSimilarity(b *testing.B) {
	s1 := "кабель медный ВВГ 3x2.5"
	s2 := "кабель медный ВВГ 3x2,5"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JaroWinklerSimilarity(s1, s2)
	}
}

// BenchmarkJaccardIndexSimilarity бенчмарк для Jaccard
func BenchmarkJaccardIndexSimilarity(b *testing.B) {
	s1 := "кабель медный ВВГ 3x2.5"
	s2 := "кабель медный ВВГ 3x2,5"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JaccardIndexSimilarity(s1, s2)
	}
}

// BenchmarkNGramSimilarity бенчмарк для N-грамм
func BenchmarkNGramSimilarity(b *testing.B) {
	s1 := "кабель медный ВВГ 3x2.5"
	s2 := "кабель медный ВВГ 3x2,5"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NGramSimilarity(s1, s2, 2)
	}
}

// BenchmarkHybridSimilarityAdvanced бенчмарк для гибридного метода
// Примечание: дубликат уже существует в advanced_similarity_test.go
// func BenchmarkHybridSimilarityAdvanced(b *testing.B) {
// 	s1 := "кабель медный ВВГ 3x2.5"
// 	s2 := "кабель медный ВВГ 3x2,5"
// 	weights := DefaultSimilarityWeights()
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		HybridSimilarityAdvanced(s1, s2, weights)
// 	}
// }

// BenchmarkPhoneticSimilarity бенчмарк для фонетических алгоритмов
func BenchmarkPhoneticSimilarity(b *testing.B) {
	s1 := "кабель медный ВВГ 3x2.5"
	s2 := "кабель медный ВВГ 3x2,5"
	pm := NewPhoneticMatcher()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.Similarity(s1, s2)
	}
}

// BenchmarkShortStrings бенчмарк для коротких строк
func BenchmarkShortStrings(b *testing.B) {
	s1 := "молоток"
	s2 := "молотак"
	b.ResetTimer()
	b.Run("Levenshtein", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			LevenshteinSimilarity(s1, s2)
		}
	})
	b.Run("JaroWinkler", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			JaroWinklerSimilarity(s1, s2)
		}
	})
	b.Run("Jaccard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			JaccardIndexSimilarity(s1, s2)
		}
	})
}

// BenchmarkLongStrings бенчмарк для длинных строк
func BenchmarkLongStrings(b *testing.B) {
	s1 := "кабель медный ВВГ 3x2.5 мм² для прокладки в помещениях и на открытом воздухе"
	s2 := "кабель медный ВВГ 3x2,5 мм² для прокладки в помещениях и на открытом воздухе"
	b.ResetTimer()
	b.Run("Levenshtein", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			LevenshteinSimilarity(s1, s2)
		}
	})
	b.Run("JaroWinkler", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			JaroWinklerSimilarity(s1, s2)
		}
	})
	b.Run("Jaccard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			JaccardIndexSimilarity(s1, s2)
		}
	})
}

