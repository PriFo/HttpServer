package algorithms

import (
	"strings"
	"sync"
)

// Lemmatizer interface defines methods for lemmatization
type Lemmatizer interface {
	// Lemmatize returns the lemma (normal form) of a word
	Lemmatize(word string) string

	// LemmatizeTokens returns lemmatized versions of multiple words
	LemmatizeTokens(tokens []string) []string

	// LemmatizeWithCache returns the lemmatized version with caching
	LemmatizeWithCache(word string) string

	// LemmatizeText lemmatizes all words in a text string
	LemmatizeText(text string) string
}

// RussianLemmatizer implements lemmatization for Russian language
// Uses rule-based approach with dictionary for common words
type RussianLemmatizer struct {
	cache    map[string]string
	mu       sync.RWMutex
	useCache bool
	// Dictionary for common words and their lemmas
	lemmaDict map[string]string
}

// NewRussianLemmatizer creates a new Russian language lemmatizer
func NewRussianLemmatizer() *RussianLemmatizer {
	lem := &RussianLemmatizer{
		cache:     make(map[string]string),
		useCache:  true,
		lemmaDict: make(map[string]string),
	}
	lem.initDictionary()
	return lem
}

// NewRussianLemmatizerWithoutCache creates a lemmatizer without caching
func NewRussianLemmatizerWithoutCache() *RussianLemmatizer {
	lem := &RussianLemmatizer{
		useCache: false,
		lemmaDict: make(map[string]string),
	}
	lem.initDictionary()
	return lem
}

// initDictionary initializes dictionary with common Russian words
func (l *RussianLemmatizer) initDictionary() {
	// Common words dictionary: word -> lemma
	commonWords := map[string]string{
		// Масло и производные
		"масла": "масло", "маслом": "масло", "масле": "масло", "маслами": "масло",
		"масло": "масло",
		
		// Сливочный и производные
		"сливочного": "сливочный", "сливочному": "сливочный", "сливочным": "сливочный",
		"сливочном": "сливочный", "сливочная": "сливочный", "сливочной": "сливочный",
		"сливочную": "сливочный", "сливочное": "сливочный", "сливочные": "сливочный",
		"сливочных": "сливочный", "сливочными": "сливочный", "сливочный": "сливочный",
		
		// Кабель и производные
		"кабеля": "кабель", "кабелю": "кабель", "кабелем": "кабель", "кабеле": "кабель",
		"кабели": "кабель", "кабелей": "кабель", "кабелям": "кабель", "кабелями": "кабель",
		"кабель": "кабель",
		
		// Молоток и производные
		"молотка": "молоток", "молотку": "молоток", "молотком": "молоток", "молотке": "молоток",
		"молотки": "молоток", "молотков": "молоток", "молоткам": "молоток", "молотками": "молоток",
		"молоток": "молоток",
		
		// Дерево и производные
		"дерева": "дерево", "дереву": "дерево", "деревом": "дерево", "дереве": "дерево",
		"деревья": "дерево", "деревьев": "дерево", "деревьям": "дерево", "деревьями": "дерево",
		"дерево": "дерево",
		
		// Сталь и производные
		"сталью": "сталь",
		"сталь": "сталь",
		
		// Пластик и производные
		"пластика": "пластик", "пластику": "пластик", "пластиком": "пластик", "пластике": "пластик",
		"пластики": "пластик", "пластиков": "пластик", "пластикам": "пластик", "пластиками": "пластик",
		"пластик": "пластик",
		
		// Белый и производные
		"белого": "белый", "белому": "белый", "белым": "белый", "белом": "белый",
		"белая": "белый", "белой": "белый", "белую": "белый", "белое": "белый",
		"белые": "белый", "белых": "белый", "белыми": "белый",
		"белый": "белый",
		
		// Черный и производные
		"черного": "черный", "черному": "черный", "черным": "черный", "черном": "черный",
		"черная": "черный", "черной": "черный", "черную": "черный", "черное": "черный",
		"черные": "черный", "черных": "черный", "черными": "черный",
		"черный": "черный",
		
		// Красный и производные
		"красного": "красный", "красному": "красный", "красным": "красный", "красном": "красный",
		"красная": "красный", "красной": "красный", "красную": "красный", "красное": "красный",
		"красные": "красный", "красных": "красный", "красными": "красный",
		"красный": "красный",
	}
	
	l.lemmaDict = commonWords
}

// Lemmatize returns the lemma (normal form) of a word
// Example: "маслами" -> "масло", "сливочного" -> "сливочный"
func (l *RussianLemmatizer) Lemmatize(word string) string {
	if word == "" {
		return ""
	}

	// Normalize to lowercase
	normalized := strings.ToLower(strings.TrimSpace(word))
	if normalized == "" {
		return ""
	}

	// Check dictionary first
	if lemma, found := l.lemmaDict[normalized]; found {
		return lemma
	}

	// Apply rule-based lemmatization
	return l.lemmatizeByRules(normalized)
}

// lemmatizeByRules applies rule-based lemmatization for Russian
func (l *RussianLemmatizer) lemmatizeByRules(word string) string {
	if len(word) < 3 {
		return word
	}

	// Common Russian endings and their replacements
	// Order matters - check longer endings first
	
	// Adjective endings (прилагательные)
	adjectiveEndings := []struct {
		ending string
		lemma  string
	}{
		{"ого", "ый"}, {"ому", "ый"}, {"ым", "ый"}, {"ом", "ый"},
		{"ой", "ый"}, {"ую", "ый"}, {"ое", "ый"}, {"ые", "ый"},
		{"ых", "ый"}, {"ыми", "ый"},
		{"его", "ий"}, {"ему", "ий"}, {"им", "ий"}, {"ем", "ий"},
		{"яя", "ий"}, {"юю", "ий"}, {"ее", "ий"}, {"ие", "ий"},
		{"их", "ий"}, {"ими", "ий"},
		{"ая", "ый"}, {"ую", "ый"}, {"ое", "ый"}, {"ые", "ый"},
		{"ых", "ый"}, {"ыми", "ый"},
	}

	for _, rule := range adjectiveEndings {
		if strings.HasSuffix(word, rule.ending) {
			stem := strings.TrimSuffix(word, rule.ending)
			if len(stem) >= 2 {
				return stem + rule.lemma
			}
		}
	}

	// Noun endings (существительные)
	nounEndings := []struct {
		ending string
		lemma  string
	}{
		// Masculine
		{"а", ""}, {"у", ""}, {"ом", ""}, {"е", ""},
		{"ы", ""}, {"ов", ""}, {"ам", ""}, {"ами", ""},
		// Feminine
		{"ы", "а"}, {"е", "а"}, {"у", "а"}, {"ой", "а"}, {"ами", "а"},
		{"ам", "а"}, {"ах", "а"},
		// Neuter
		{"а", "о"}, {"у", "о"}, {"ом", "о"}, {"е", "о"},
		{"ы", "о"}, {"ов", "о"}, {"ам", "о"}, {"ами", "о"},
	}

	for _, rule := range nounEndings {
		if strings.HasSuffix(word, rule.ending) {
			stem := strings.TrimSuffix(word, rule.ending)
			if len(stem) >= 2 {
				if rule.lemma != "" {
					return stem + rule.lemma
				}
				return stem
			}
		}
	}

	// Verb endings (глаголы) - simplified
	verbEndings := []struct {
		ending string
		lemma  string
	}{
		{"ю", "ть"}, {"ешь", "ть"}, {"ет", "ть"}, {"ем", "ть"},
		{"ете", "ть"}, {"ут", "ть"}, {"ют", "ть"},
		{"ал", "ть"}, {"ала", "ть"}, {"ало", "ть"}, {"али", "ть"},
		{"ил", "ть"}, {"ила", "ть"}, {"ило", "ть"}, {"или", "ть"},
	}

	for _, rule := range verbEndings {
		if strings.HasSuffix(word, rule.ending) {
			stem := strings.TrimSuffix(word, rule.ending)
			if len(stem) >= 2 {
				return stem + rule.lemma
			}
		}
	}

	// If no rule matches, return the word as-is
	return word
}

// LemmatizeWithCache returns the lemmatized version with caching
func (l *RussianLemmatizer) LemmatizeWithCache(word string) string {
	if !l.useCache {
		return l.Lemmatize(word)
	}

	normalized := strings.ToLower(strings.TrimSpace(word))
	if normalized == "" {
		return ""
	}

	// Check cache first
	l.mu.RLock()
	if cached, found := l.cache[normalized]; found {
		l.mu.RUnlock()
		return cached
	}
	l.mu.RUnlock()

	// Lemmatize the word
	lemma := l.Lemmatize(normalized)

	// Store in cache
	l.mu.Lock()
	l.cache[normalized] = lemma
	l.mu.Unlock()

	return lemma
}

// LemmatizeTokens returns lemmatized versions of multiple words
// Example: ["маслами", "сливочного"] -> ["масло", "сливочный"]
func (l *RussianLemmatizer) LemmatizeTokens(tokens []string) []string {
	if len(tokens) == 0 {
		return []string{}
	}

	lemmatized := make([]string, len(tokens))
	for i, token := range tokens {
		lemmatized[i] = l.LemmatizeWithCache(token)
	}

	return lemmatized
}

// LemmatizeText lemmatizes all words in a text string
// Example: "маслами сливочного" -> "масло сливочный"
func (l *RussianLemmatizer) LemmatizeText(text string) string {
	if text == "" {
		return ""
	}

	// Split into words
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	// Lemmatize each word
	lemmatized := l.LemmatizeTokens(words)

	// Join back
	return strings.Join(lemmatized, " ")
}

// ClearCache clears the internal cache
func (l *RussianLemmatizer) ClearCache() {
	if !l.useCache {
		return
	}

	l.mu.Lock()
	l.cache = make(map[string]string)
	l.mu.Unlock()
}

// GetCacheSize returns the number of cached items
func (l *RussianLemmatizer) GetCacheSize() int {
	if !l.useCache {
		return 0
	}

	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.cache)
}

// AddToDictionary adds a word-lemma pair to the dictionary
func (l *RussianLemmatizer) AddToDictionary(word, lemma string) {
	normalized := strings.ToLower(strings.TrimSpace(word))
	if normalized != "" {
		l.lemmaDict[normalized] = strings.ToLower(strings.TrimSpace(lemma))
	}
}

// LemmatizeSimilarity calculates similarity between two words based on their lemmas
// Returns 1.0 if lemmas are identical, 0.0 if completely different
func (l *RussianLemmatizer) LemmatizeSimilarity(word1, word2 string) float64 {
	lemma1 := l.LemmatizeWithCache(word1)
	lemma2 := l.LemmatizeWithCache(word2)

	if lemma1 == "" && lemma2 == "" {
		return 1.0
	}

	if lemma1 == "" || lemma2 == "" {
		return 0.0
	}

	if lemma1 == lemma2 {
		return 1.0
	}

	return 0.0
}

// BatchLemmatize processes multiple texts in parallel
func (l *RussianLemmatizer) BatchLemmatize(texts []string, workers int) []string {
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
				results[idx] = l.LemmatizeText(texts[idx])
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

