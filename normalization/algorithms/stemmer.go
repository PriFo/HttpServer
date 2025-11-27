package algorithms

import (
	"strings"
	"sync"

	"github.com/kljensen/snowball"
)

// Stemmer interface defines methods for stemming words
type Stemmer interface {
	// Stem returns the stemmed version of a word
	Stem(word string) string

	// StemTokens returns stemmed versions of multiple words
	StemTokens(tokens []string) []string

	// StemWithCache returns the stemmed version with caching
	StemWithCache(word string) string
}

// RussianStemmer implements stemming for Russian language using Snowball algorithm
type RussianStemmer struct {
	language string
	cache    map[string]string
	mu       sync.RWMutex
	useCache bool
}

// NewRussianStemmer creates a new Russian language stemmer
func NewRussianStemmer() *RussianStemmer {
	return &RussianStemmer{
		language: "russian",
		cache:    make(map[string]string),
		useCache: true,
	}
}

// NewRussianStemmerWithoutCache creates a stemmer without caching
func NewRussianStemmerWithoutCache() *RussianStemmer {
	return &RussianStemmer{
		language: "russian",
		useCache: false,
	}
}

// Stem returns the stemmed version of a word using Snowball algorithm
// Example: "молотком" -> "молот", "кабеля" -> "кабел"
func (s *RussianStemmer) Stem(word string) string {
	if word == "" {
		return ""
	}

	// Normalize to lowercase for consistency
	normalized := strings.ToLower(strings.TrimSpace(word))

	if normalized == "" {
		return ""
	}

	// Use Snowball stemmer
	stemmed, err := snowball.Stem(normalized, s.language, true)
	if err != nil {
		// If stemming fails, return the normalized word
		return normalized
	}

	return stemmed
}

// StemWithCache returns the stemmed version with caching for performance
func (s *RussianStemmer) StemWithCache(word string) string {
	if !s.useCache {
		return s.Stem(word)
	}

	normalized := strings.ToLower(strings.TrimSpace(word))
	if normalized == "" {
		return ""
	}

	// Check cache first
	s.mu.RLock()
	if cached, found := s.cache[normalized]; found {
		s.mu.RUnlock()
		return cached
	}
	s.mu.RUnlock()

	// Stem the word
	stemmed := s.Stem(normalized)

	// Store in cache
	s.mu.Lock()
	s.cache[normalized] = stemmed
	s.mu.Unlock()

	return stemmed
}

// StemTokens returns stemmed versions of multiple words
// Example: ["молоток", "молотка", "молотком"] -> ["молот", "молот", "молот"]
func (s *RussianStemmer) StemTokens(tokens []string) []string {
	if len(tokens) == 0 {
		return []string{}
	}

	stemmed := make([]string, len(tokens))
	for i, token := range tokens {
		stemmed[i] = s.StemWithCache(token)
	}

	return stemmed
}

// StemText stems all words in a text string and returns the stemmed text
// Example: "красный молоток и синий молотка" -> "красн молот и син молот"
func (s *RussianStemmer) StemText(text string) string {
	if text == "" {
		return ""
	}

	// Split into words
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	// Stem each word
	stemmed := s.StemTokens(words)

	// Join back
	return strings.Join(stemmed, " ")
}

// ClearCache clears the internal cache
func (s *RussianStemmer) ClearCache() {
	if !s.useCache {
		return
	}

	s.mu.Lock()
	s.cache = make(map[string]string)
	s.mu.Unlock()
}

// GetCacheSize returns the number of cached items
func (s *RussianStemmer) GetCacheSize() int {
	if !s.useCache {
		return 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cache)
}

// StemSimilarity calculates similarity between two words based on their stems
// Returns 1.0 if stems are identical, 0.0 if completely different
func (s *RussianStemmer) StemSimilarity(word1, word2 string) float64 {
	stem1 := s.StemWithCache(word1)
	stem2 := s.StemWithCache(word2)

	if stem1 == "" && stem2 == "" {
		return 1.0
	}

	if stem1 == "" || stem2 == "" {
		return 0.0
	}

	if stem1 == stem2 {
		return 1.0
	}

	return 0.0
}

// BatchStem processes multiple texts in parallel
func (s *RussianStemmer) BatchStem(texts []string, workers int) []string {
	if len(texts) == 0 {
		return []string{}
	}

	if workers <= 0 {
		workers = 4
	}

	results := make([]string, len(texts))
	var wg sync.WaitGroup
	jobs := make(chan int, len(texts))

	// Start worker goroutines
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results[idx] = s.StemText(texts[idx])
			}
		}()
	}

	// Send jobs
	for i := range texts {
		jobs <- i
	}
	close(jobs)

	// Wait for completion
	wg.Wait()

	return results
}

// GetCommonStem returns the common stem if all words share the same stem, empty string otherwise
// Example: ["молоток", "молотка", "молотком"] -> "молот"
func (s *RussianStemmer) GetCommonStem(words []string) string {
	if len(words) == 0 {
		return ""
	}

	if len(words) == 1 {
		return s.StemWithCache(words[0])
	}

	firstStem := s.StemWithCache(words[0])
	for i := 1; i < len(words); i++ {
		if s.StemWithCache(words[i]) != firstStem {
			return ""
		}
	}

	return firstStem
}
