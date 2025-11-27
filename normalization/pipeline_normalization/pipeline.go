package pipeline_normalization

import (
	"fmt"
	"sync"
	"time"

	"httpserver/normalization/algorithms"
)

// AlgorithmExecutor интерфейс для выполнения алгоритма
type AlgorithmExecutor interface {
	Execute(text1, text2 string, config AlgorithmConfig) (float64, error)
	GetName() string
}

// NormalizationPipeline основной pipeline для нормализации
type NormalizationPipeline struct {
	config     *NormalizationPipelineConfig
	executors  map[AlgorithmType]AlgorithmExecutor
	cache      map[string]*NormalizationResult
	cacheMutex sync.RWMutex
}

// NewNormalizationPipeline создает новый pipeline нормализации
func NewNormalizationPipeline(config *NormalizationPipelineConfig) (*NormalizationPipeline, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	pipeline := &NormalizationPipeline{
		config:    config,
		executors: make(map[AlgorithmType]AlgorithmExecutor),
		cache:     make(map[string]*NormalizationResult),
	}

	// Инициализируем исполнители алгоритмов
	if err := pipeline.initializeExecutors(); err != nil {
		return nil, fmt.Errorf("failed to initialize executors: %w", err)
	}

	return pipeline, nil
}

// initializeExecutors инициализирует исполнители алгоритмов
func (p *NormalizationPipeline) initializeExecutors() error {
	for _, algConfig := range p.config.Algorithms {
		if !algConfig.Enabled {
			continue
		}

		var executor AlgorithmExecutor
		var err error

		switch algConfig.Type {
		case AlgorithmSoundex:
			executor = NewSoundexExecutor(algConfig)
		case AlgorithmMetaphone:
			executor = NewMetaphoneExecutor(algConfig)
		case AlgorithmJaccard:
			executor, err = NewJaccardExecutor(algConfig)
		case AlgorithmNGrams:
			executor, err = NewNGramsExecutor(algConfig)
		case AlgorithmDamerauLevenshtein:
			executor = NewDamerauLevenshteinExecutor(algConfig)
		case AlgorithmCosine:
			executor, err = NewCosineExecutor(algConfig)
		case AlgorithmToken:
			executor, err = NewTokenExecutor(algConfig)
		case AlgorithmJaro:
			executor = NewJaroExecutor(algConfig)
		case AlgorithmJaroWinkler:
			executor = NewJaroWinklerExecutor(algConfig)
		case AlgorithmLCS:
			executor = NewLCSExecutor(algConfig)
		default:
			return fmt.Errorf("unknown algorithm type: %s", algConfig.Type)
		}

		if err != nil {
			return fmt.Errorf("failed to create executor for %s: %w", algConfig.Type, err)
		}

		p.executors[algConfig.Type] = executor
	}

	return nil
}

// Normalize вычисляет схожесть двух строк
func (p *NormalizationPipeline) Normalize(text1, text2 string) (*NormalizationResult, error) {
	startTime := time.Now()

	// Проверяем кэш
	if p.config.CacheEnabled {
		if cached := p.getFromCache(text1, text2); cached != nil {
			return cached, nil
		}
	}

	// Создаем результат
	result := &NormalizationResult{
		Text1:           text1,
		Text2:           text2,
		NormalizedText1: text1, // Будет нормализовано при необходимости
		NormalizedText2: text2,
		Similarity:      *NewSimilarityScore(),
		AlgorithmsUsed:  make([]string, 0),
	}

	// Выполняем алгоритмы
	if p.config.ParallelExecution {
		p.executeAlgorithmsParallel(text1, text2, result)
	} else {
		p.executeAlgorithmsSequential(text1, text2, result)
	}

	// Вычисляем общую схожесть
	weights := p.getWeights()
	result.Similarity.CalculateOverall(weights, p.config.CombineMethod, p.config.MinSimilarity)

	// Записываем время обработки
	result.ProcessingTime = time.Since(startTime).Nanoseconds()

	// Сохраняем в кэш
	if p.config.CacheEnabled {
		p.saveToCache(text1, text2, result)
	}

	return result, nil
}

// executeAlgorithmsSequential выполняет алгоритмы последовательно
func (p *NormalizationPipeline) executeAlgorithmsSequential(text1, text2 string, result *NormalizationResult) {
	for _, algConfig := range p.config.GetEnabledAlgorithms() {
		executor, exists := p.executors[algConfig.Type]
		if !exists {
			continue
		}

		score, err := executor.Execute(text1, text2, algConfig)
		if err != nil {
			continue // Пропускаем алгоритм при ошибке
		}

		// Добавляем оценку только если она превышает порог
		if score >= algConfig.Threshold {
			result.Similarity.AddAlgorithmScore(executor.GetName(), score)
			result.AlgorithmsUsed = append(result.AlgorithmsUsed, executor.GetName())
		}
	}
}

// executeAlgorithmsParallel выполняет алгоритмы параллельно
func (p *NormalizationPipeline) executeAlgorithmsParallel(text1, text2 string, result *NormalizationResult) {
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, algConfig := range p.config.GetEnabledAlgorithms() {
		executor, exists := p.executors[algConfig.Type]
		if !exists {
			continue
		}

		wg.Add(1)
		go func(exec AlgorithmExecutor, config AlgorithmConfig) {
			defer wg.Done()

			score, err := exec.Execute(text1, text2, config)
			if err != nil {
				return
			}

			if score >= config.Threshold {
				mutex.Lock()
				result.Similarity.AddAlgorithmScore(exec.GetName(), score)
				result.AlgorithmsUsed = append(result.AlgorithmsUsed, exec.GetName())
				mutex.Unlock()
			}
		}(executor, algConfig)
	}

	wg.Wait()
}

// getWeights возвращает веса алгоритмов
func (p *NormalizationPipeline) getWeights() map[string]float64 {
	weights := make(map[string]float64)
	for _, algConfig := range p.config.GetEnabledAlgorithms() {
		executor, exists := p.executors[algConfig.Type]
		if exists {
			weights[executor.GetName()] = algConfig.Weight
		}
	}
	return weights
}

// getFromCache получает результат из кэша
func (p *NormalizationPipeline) getFromCache(text1, text2 string) *NormalizationResult {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	key := p.makeCacheKey(text1, text2)
	if result, exists := p.cache[key]; exists {
		return result
	}
	return nil
}

// saveToCache сохраняет результат в кэш
func (p *NormalizationPipeline) saveToCache(text1, text2 string, result *NormalizationResult) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	key := p.makeCacheKey(text1, text2)
	p.cache[key] = result
}

// makeCacheKey создает ключ для кэша
func (p *NormalizationPipeline) makeCacheKey(text1, text2 string) string {
	// Используем упорядоченную пару для симметричного кэша
	if text1 > text2 {
		text1, text2 = text2, text1
	}
	return fmt.Sprintf("%s|%s", text1, text2)
}

// ClearCache очищает кэш
func (p *NormalizationPipeline) ClearCache() {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	p.cache = make(map[string]*NormalizationResult)
}

// BatchNormalize обрабатывает батч пар строк
func (p *NormalizationPipeline) BatchNormalize(pairs [][]string) (*BatchResult, error) {
	startTime := time.Now()
	batchResult := &BatchResult{
		Results:        make([]NormalizationResult, 0, len(pairs)),
		TotalProcessed: len(pairs),
		DuplicatesFound: 0,
	}

	for _, pair := range pairs {
		if len(pair) != 2 {
			continue
		}

		result, err := p.Normalize(pair[0], pair[1])
		if err != nil {
			continue
		}

		batchResult.Results = append(batchResult.Results, *result)
		if result.Similarity.IsDuplicate {
			batchResult.DuplicatesFound++
		}
	}

	batchResult.ProcessingTime = time.Since(startTime).Nanoseconds()

	// Вычисляем среднюю схожесть
	if len(batchResult.Results) > 0 {
		sum := 0.0
		for _, result := range batchResult.Results {
			sum += result.Similarity.OverallSimilarity
		}
		batchResult.AverageSimilarity = sum / float64(len(batchResult.Results))
	}

	return batchResult, nil
}

// GetConfig возвращает конфигурацию pipeline
func (p *NormalizationPipeline) GetConfig() *NormalizationPipelineConfig {
	return p.config
}

// UpdateConfig обновляет конфигурацию pipeline
func (p *NormalizationPipeline) UpdateConfig(config *NormalizationPipelineConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	p.config = config
	p.ClearCache()

	// Переинициализируем исполнители
	return p.initializeExecutors()
}

// --- Реализации исполнителей алгоритмов ---

// SoundexExecutor исполнитель для Soundex
type SoundexExecutor struct {
	config AlgorithmConfig
	soundex *algorithms.SoundexRU
}

func NewSoundexExecutor(config AlgorithmConfig) *SoundexExecutor {
	return &SoundexExecutor{
		config:  config,
		soundex: algorithms.NewSoundexRU(),
	}
}

func (e *SoundexExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return e.soundex.Similarity(text1, text2), nil
}

func (e *SoundexExecutor) GetName() string {
	return "soundex"
}

// MetaphoneExecutor исполнитель для Metaphone
type MetaphoneExecutor struct {
	config    AlgorithmConfig
	metaphone *algorithms.MetaphoneRU
}

func NewMetaphoneExecutor(config AlgorithmConfig) *MetaphoneExecutor {
	return &MetaphoneExecutor{
		config:    config,
		metaphone: algorithms.NewMetaphoneRU(),
	}
}

func (e *MetaphoneExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return e.metaphone.Similarity(text1, text2), nil
}

func (e *MetaphoneExecutor) GetName() string {
	return "metaphone"
}

// JaccardExecutor исполнитель для Jaccard
type JaccardExecutor struct {
	config  AlgorithmConfig
	jaccard *algorithms.JaccardIndex
}

func NewJaccardExecutor(config AlgorithmConfig) (*JaccardExecutor, error) {
	useNGrams := false
	if val, ok := config.Params["use_ngrams"].(bool); ok {
		useNGrams = val
	}

	var jaccard *algorithms.JaccardIndex
	if useNGrams {
		nGramSize := 2
		if val, ok := config.Params["n_gram_size"].(int); ok {
			nGramSize = val
		} else if val, ok := config.Params["n_gram_size"].(float64); ok {
			nGramSize = int(val)
		}
		jaccard = algorithms.NewJaccardIndexWithNGrams(nGramSize)
	} else {
		jaccard = algorithms.NewJaccardIndex()
	}

	return &JaccardExecutor{
		config:  config,
		jaccard: jaccard,
	}, nil
}

func (e *JaccardExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return e.jaccard.Similarity(text1, text2), nil
}

func (e *JaccardExecutor) GetName() string {
	return "jaccard"
}

// NGramsExecutor исполнитель для N-грамм
type NGramsExecutor struct {
	config  AlgorithmConfig
	ngrams  *algorithms.NGramGenerator
}

func NewNGramsExecutor(config AlgorithmConfig) (*NGramsExecutor, error) {
	nGramSize := 2
	if val, ok := config.Params["n"].(int); ok {
		nGramSize = val
	} else if val, ok := config.Params["n"].(float64); ok {
		nGramSize = int(val)
	}

	return &NGramsExecutor{
		config: config,
		ngrams:  algorithms.NewNGramGenerator(nGramSize),
	}, nil
}

func (e *NGramsExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return e.ngrams.Similarity(text1, text2), nil
}

func (e *NGramsExecutor) GetName() string {
	return "ngrams"
}

// DamerauLevenshteinExecutor исполнитель для Дамерау-Левенштейна
type DamerauLevenshteinExecutor struct {
	config            AlgorithmConfig
	damerauLevenshtein *algorithms.DamerauLevenshtein
}

func NewDamerauLevenshteinExecutor(config AlgorithmConfig) *DamerauLevenshteinExecutor {
	return &DamerauLevenshteinExecutor{
		config:            config,
		damerauLevenshtein: algorithms.NewDamerauLevenshtein(),
	}
}

func (e *DamerauLevenshteinExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return e.damerauLevenshtein.Similarity(text1, text2), nil
}

func (e *DamerauLevenshteinExecutor) GetName() string {
	return "damerau_levenshtein"
}

// CosineExecutor исполнитель для косинусной близости
type CosineExecutor struct {
	config  AlgorithmConfig
	cosine  *algorithms.CosineSimilarity
}

func NewCosineExecutor(config AlgorithmConfig) (*CosineExecutor, error) {
	useTFIDF := true
	if val, ok := config.Params["use_tfidf"].(bool); ok {
		useTFIDF = val
	}

	var cosine *algorithms.CosineSimilarity
	if useTFIDF {
		cosine = algorithms.NewCosineSimilarity()
	} else {
		useBinary := false
		if val, ok := config.Params["use_binary"].(bool); ok {
			useBinary = val
		}
		if useBinary {
			cosine = algorithms.NewCosineSimilarityBinary()
		} else {
			cosine = algorithms.NewCosineSimilarityFrequency()
		}
	}

	return &CosineExecutor{
		config: config,
		cosine: cosine,
	}, nil
}

func (e *CosineExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return e.cosine.Similarity(text1, text2), nil
}

func (e *CosineExecutor) GetName() string {
	return "cosine"
}

// TokenExecutor исполнитель для токен-ориентированных методов
type TokenExecutor struct {
	config  AlgorithmConfig
	token   *algorithms.TokenBasedSimilarity
}

func NewTokenExecutor(config AlgorithmConfig) (*TokenExecutor, error) {
	useWeighted := false
	if val, ok := config.Params["use_weighted"].(bool); ok {
		useWeighted = val
	}

	var token *algorithms.TokenBasedSimilarity
	if useWeighted {
		token = algorithms.NewTokenBasedSimilarityWeighted()
	} else {
		token = algorithms.NewTokenBasedSimilarity()
	}

	return &TokenExecutor{
		config: config,
		token:  token,
	}, nil
}

func (e *TokenExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return e.token.Similarity(text1, text2), nil
}

func (e *TokenExecutor) GetName() string {
	return "token"
}

// JaroExecutor исполнитель для алгоритма Jaro
type JaroExecutor struct {
	config AlgorithmConfig
}

func NewJaroExecutor(config AlgorithmConfig) *JaroExecutor {
	return &JaroExecutor{
		config: config,
	}
}

func (e *JaroExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return algorithms.JaroSimilarityAdvanced(text1, text2), nil
}

func (e *JaroExecutor) GetName() string {
	return "jaro"
}

// JaroWinklerExecutor исполнитель для алгоритма Jaro-Winkler
type JaroWinklerExecutor struct {
	config AlgorithmConfig
}

func NewJaroWinklerExecutor(config AlgorithmConfig) *JaroWinklerExecutor {
	return &JaroWinklerExecutor{
		config: config,
	}
}

func (e *JaroWinklerExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return algorithms.JaroWinklerSimilarityAdvanced(text1, text2), nil
}

func (e *JaroWinklerExecutor) GetName() string {
	return "jaro_winkler"
}

// LCSExecutor исполнитель для алгоритма LCS
type LCSExecutor struct {
	config AlgorithmConfig
}

func NewLCSExecutor(config AlgorithmConfig) *LCSExecutor {
	return &LCSExecutor{
		config: config,
	}
}

func (e *LCSExecutor) Execute(text1, text2 string, config AlgorithmConfig) (float64, error) {
	return algorithms.LCSSimilarityAdvanced(text1, text2), nil
}

func (e *LCSExecutor) GetName() string {
	return "lcs"
}

