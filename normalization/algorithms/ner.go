package algorithms

import (
	"regexp"
	"strings"
)

// NEREntityType тип именованной сущности
type NEREntityType string

const (
	// MATERIAL - материал (сталь, дерево, пластик, медь)
	NEREntityTypeMaterial NEREntityType = "MATERIAL"
	// SIZE - размер (20x30, 100мм)
	NEREntityTypeSize NEREntityType = "SIZE"
	// COLOR - цвет (белый, черный, красный)
	NEREntityTypeColor NEREntityType = "COLOR"
	// TYPE - тип (многожильный, одножильный)
	NEREntityTypeType NEREntityType = "TYPE"
	// DIMENSION - размерность (ширина, высота, глубина)
	NEREntityTypeDimension NEREntityType = "DIMENSION"
	// WEIGHT - вес (кг, г)
	NEREntityTypeWeight NEREntityType = "WEIGHT"
	// LENGTH - длина (м, см, мм)
	NEREntityTypeLength NEREntityType = "LENGTH"
	// VOLUME - объем (л, мл)
	NEREntityTypeVolume NEREntityType = "VOLUME"
	// POWER - мощность (вт, квт)
	NEREntityTypePower NEREntityType = "POWER"
	// CODE - код/артикул
	NEREntityTypeCode NEREntityType = "CODE"
)

// BIOTag BIO-тег для NER
type BIOTag string

const (
	BIO_B BIOTag = "B" // Beginning - начало сущности
	BIO_I BIOTag = "I" // Inside - внутри сущности
	BIO_O BIOTag = "O" // Outside - вне сущности
)

// NEREntity представляет именованную сущность
type NEREntity struct {
	Type       NEREntityType `json:"type"`        // Тип сущности
	Text       string        `json:"text"`        // Текст сущности
	Start      int           `json:"start"`       // Начальная позиция
	End        int           `json:"end"`         // Конечная позиция
	Confidence float64       `json:"confidence"`  // Уверенность (0.0-1.0)
	Value      string        `json:"value"`       // Нормализованное значение
	Unit       string        `json:"unit"`        // Единица измерения (если есть)
}

// BIOTaggedToken токен с BIO-тегом
type BIOTaggedToken struct {
	Token     string        `json:"token"`
	Tag       BIOTag        `json:"tag"`
	EntityType NEREntityType `json:"entity_type"`
	Start     int           `json:"start"`
	End       int           `json:"end"`
}

// RussianNER реализует Named Entity Recognition для русского языка
type RussianNER struct {
	// Словари для распознавания
	materials map[string]string  // слово -> нормализованное значение
	colors    map[string]string
	types     map[string]string
	
	// Regex паттерны
	dimensionRegex     *regexp.Regexp
	sizeRegex          *regexp.Regexp
	weightRegex        *regexp.Regexp
	lengthRegex        *regexp.Regexp
	volumeRegex        *regexp.Regexp
	powerRegex         *regexp.Regexp
	codeRegex          *regexp.Regexp
}

// NewRussianNER создает новый NER для русского языка
func NewRussianNER() *RussianNER {
	ner := &RussianNER{
		materials: make(map[string]string),
		colors:    make(map[string]string),
		types:     make(map[string]string),
	}
	
	ner.initDictionaries()
	ner.initRegexPatterns()
	
	return ner
}

// initDictionaries инициализирует словари для распознавания
func (ner *RussianNER) initDictionaries() {
	// Материалы
	materials := map[string]string{
		"сталь": "сталь", "стальной": "сталь", "стальная": "сталь", "стальное": "сталь",
		"дерево": "дерево", "деревянный": "дерево", "деревянная": "дерево", "деревянное": "дерево",
		"пластик": "пластик", "пластиковый": "пластик", "пластиковая": "пластик", "пластиковое": "пластик",
		"медь": "медь", "медный": "медь", "медная": "медь", "медное": "медь",
		"алюминий": "алюминий", "алюминиевый": "алюминий", "алюминиевая": "алюминий", "алюминиевое": "алюминий",
		"железо": "железо", "железный": "железо", "железная": "железо", "железное": "железо",
		"стекло": "стекло", "стеклянный": "стекло", "стеклянная": "стекло", "стеклянное": "стекло",
		"резина": "резина", "резиновый": "резина", "резиновая": "резина", "резиновое": "резина",
		"кожа": "кожа", "кожаный": "кожа", "кожаная": "кожа", "кожаное": "кожа",
		"ткань": "ткань", "тканевый": "ткань", "тканевая": "ткань", "тканевое": "ткань",
	}
	
	// Цвета
	colors := map[string]string{
		"белый": "белый", "белая": "белый", "белое": "белый", "белые": "белый",
		"черный": "черный", "черная": "черный", "черное": "черный", "черные": "черный",
		"серый": "серый", "серая": "серый", "серое": "серый", "серые": "серый",
		"красный": "красный", "красная": "красный", "красное": "красный", "красные": "красный",
		"синий": "синий", "синяя": "синий", "синее": "синий", "синие": "синий",
		"зеленый": "зеленый", "зеленая": "зеленый", "зеленое": "зеленый", "зеленые": "зеленый",
		"желтый": "желтый", "желтая": "желтый", "желтое": "желтый", "желтые": "желтый",
		"коричневый": "коричневый", "коричневая": "коричневый", "коричневое": "коричневый", "коричневые": "коричневый",
		"оранжевый": "оранжевый", "оранжевая": "оранжевый", "оранжевое": "оранжевый", "оранжевые": "оранжевый",
		"фиолетовый": "фиолетовый", "фиолетовая": "фиолетовый", "фиолетовое": "фиолетовый", "фиолетовые": "фиолетовый",
		"розовый": "розовый", "розовая": "розовый", "розовое": "розовый", "розовые": "розовый",
	}
	
	// Типы
	types := map[string]string{
		"многожильный": "многожильный", "многожильная": "многожильный", "многожильное": "многожильный",
		"одножильный": "одножильный", "одножильная": "одножильный", "одножильное": "одножильный",
		"двухжильный": "двухжильный", "двухжильная": "двухжильный", "двухжильное": "двухжильный",
		"трехжильный": "трехжильный", "трехжильная": "трехжильный", "трехжильное": "трехжильный",
		"четырехжильный": "четырехжильный", "четырехжильная": "четырехжильный", "четырехжильное": "четырехжильный",
		"пятижильный": "пятижильный", "пятижильная": "пятижильный", "пятижильное": "пятижильный",
	}
	
	ner.materials = materials
	ner.colors = colors
	ner.types = types
}

// initRegexPatterns инициализирует regex паттерны
func (ner *RussianNER) initRegexPatterns() {
	// Размеры вида 100x100 или 100х100
	ner.dimensionRegex = regexp.MustCompile(`\d+[xх]\d+`)
	
	// Размеры с единицами (20x30мм, 100x200см)
	ner.sizeRegex = regexp.MustCompile(`(\d+[xх]\d+)\s*(mm|cm|m|см|мм|м)`)
	
	// Вес (кг, г, мг)
	ner.weightRegex = regexp.MustCompile(`(\d+\.?\d*)\s*(kg|g|мг|кг|г)`)
	
	// Длина (м, см, мм)
	ner.lengthRegex = regexp.MustCompile(`(\d+\.?\d*)\s*(m|cm|mm|м|см|мм)`)
	
	// Объем (л, мл)
	ner.volumeRegex = regexp.MustCompile(`(\d+\.?\d*)\s*(l|ml|л|мл)`)
	
	// Мощность (вт, квт)
	ner.powerRegex = regexp.MustCompile(`(\d+\.?\d*)\s*(w|watt|kw|вт|квт)`)
	
	// Коды/артикулы (улучшенный паттерн)
	ner.codeRegex = regexp.MustCompile(`\b[A-Z]{2}-\d+\b|[A-Z]{2}-\d+`)
}

// ExtractEntities извлекает именованные сущности из текста
func (ner *RussianNER) ExtractEntities(text string) []NEREntity {
	if text == "" {
		return nil
	}
	
	var entities []NEREntity
	normalized := strings.ToLower(strings.TrimSpace(text))
	original := strings.TrimSpace(text) // Сохраняем оригинал для кодов
	
	// 1. Извлекаем материалы
	entities = append(entities, ner.extractMaterials(normalized)...)
	
	// 2. Извлекаем цвета
	entities = append(entities, ner.extractColors(normalized)...)
	
	// 3. Извлекаем типы
	entities = append(entities, ner.extractTypes(normalized)...)
	
	// 4. Извлекаем размеры
	entities = append(entities, ner.extractDimensions(normalized)...)
	
	// 5. Извлекаем веса
	entities = append(entities, ner.extractWeights(normalized)...)
	
	// 6. Извлекаем длины
	entities = append(entities, ner.extractLengths(normalized)...)
	
	// 7. Извлекаем объемы
	entities = append(entities, ner.extractVolumes(normalized)...)
	
	// 8. Извлекаем мощности
	entities = append(entities, ner.extractPowers(normalized)...)
	
	// 9. Извлекаем коды (используем оригинальный текст, так как коды могут быть в верхнем регистре)
	entities = append(entities, ner.extractCodes(original)...)
	
	// Удаляем перекрывающиеся сущности (оставляем с большей уверенностью)
	entities = ner.removeOverlapping(entities)
	
	return entities
}

// extractMaterials извлекает материалы
func (ner *RussianNER) extractMaterials(text string) []NEREntity {
	var entities []NEREntity
	words := strings.Fields(text)
	pos := 0
	
	for _, word := range words {
		// Ищем слово в тексте начиная с позиции pos
		wordStart := strings.Index(text[pos:], word)
		if wordStart < 0 {
			continue
		}
		wordStart += pos
		
		// Проверяем все варианты слова (с учетом окончаний)
		wordLower := strings.ToLower(word)
		if material, found := ner.materials[wordLower]; found {
			entities = append(entities, NEREntity{
				Type:       NEREntityTypeMaterial,
				Text:       word,
				Start:      wordStart,
				End:        wordStart + len(word),
				Confidence: 0.9,
				Value:      material,
			})
		}
		
		pos = wordStart + len(word)
	}
	
	return entities
}

// extractColors извлекает цвета
func (ner *RussianNER) extractColors(text string) []NEREntity {
	var entities []NEREntity
	words := strings.Fields(text)
	pos := 0
	
	for _, word := range words {
		wordStart := strings.Index(text[pos:], word)
		if wordStart < 0 {
			continue
		}
		wordStart += pos
		wordLower := strings.ToLower(word)
		
		if color, found := ner.colors[wordLower]; found {
			entities = append(entities, NEREntity{
				Type:       NEREntityTypeColor,
				Text:       word,
				Start:      wordStart,
				End:        wordStart + len(word),
				Confidence: 0.9,
				Value:      color,
			})
		}
		
		pos = wordStart + len(word)
	}
	
	return entities
}

// extractTypes извлекает типы
func (ner *RussianNER) extractTypes(text string) []NEREntity {
	var entities []NEREntity
	words := strings.Fields(text)
	pos := 0
	
	for _, word := range words {
		wordStart := strings.Index(text[pos:], word)
		if wordStart < 0 {
			continue
		}
		wordStart += pos
		wordLower := strings.ToLower(word)
		
		if typ, found := ner.types[wordLower]; found {
			entities = append(entities, NEREntity{
				Type:       NEREntityTypeType,
				Text:       word,
				Start:      wordStart,
				End:        wordStart + len(word),
				Confidence: 0.85,
				Value:      typ,
			})
		}
		
		pos = wordStart + len(word)
	}
	
	return entities
}

// extractDimensions извлекает размеры
func (ner *RussianNER) extractDimensions(text string) []NEREntity {
	var entities []NEREntity
	matches := ner.dimensionRegex.FindAllStringSubmatchIndex(text, -1)
	
	for _, match := range matches {
		if len(match) >= 2 {
			start, end := match[0], match[1]
			dimText := text[start:end]
			parts := regexp.MustCompile(`[xх]`).Split(dimText, -1)
			
			if len(parts) >= 2 {
				entities = append(entities, NEREntity{
					Type:       NEREntityTypeDimension,
					Text:       dimText,
					Start:      start,
					End:        end,
					Confidence: 1.0,
					Value:      dimText,
				})
			}
		}
	}
	
	return entities
}

// extractWeights извлекает веса
func (ner *RussianNER) extractWeights(text string) []NEREntity {
	var entities []NEREntity
	matches := ner.weightRegex.FindAllStringSubmatchIndex(text, -1)
	
	for _, match := range matches {
		if len(match) >= 4 {
			start, end := match[0], match[1]
			value := text[match[2]:match[3]]
			unit := text[match[4]:match[5]]
			
			entities = append(entities, NEREntity{
				Type:       NEREntityTypeWeight,
				Text:       text[start:end],
				Start:      start,
				End:        end,
				Confidence: 1.0,
				Value:      value,
				Unit:       unit,
			})
		}
	}
	
	return entities
}

// extractLengths извлекает длины
func (ner *RussianNER) extractLengths(text string) []NEREntity {
	var entities []NEREntity
	matches := ner.lengthRegex.FindAllStringSubmatchIndex(text, -1)
	
	for _, match := range matches {
		if len(match) >= 4 {
			start, end := match[0], match[1]
			value := text[match[2]:match[3]]
			unit := text[match[4]:match[5]]
			
			entities = append(entities, NEREntity{
				Type:       NEREntityTypeLength,
				Text:       text[start:end],
				Start:      start,
				End:        end,
				Confidence: 1.0,
				Value:      value,
				Unit:       unit,
			})
		}
	}
	
	return entities
}

// extractVolumes извлекает объемы
func (ner *RussianNER) extractVolumes(text string) []NEREntity {
	var entities []NEREntity
	matches := ner.volumeRegex.FindAllStringSubmatchIndex(text, -1)
	
	for _, match := range matches {
		if len(match) >= 4 {
			start, end := match[0], match[1]
			value := text[match[2]:match[3]]
			unit := text[match[4]:match[5]]
			
			entities = append(entities, NEREntity{
				Type:       NEREntityTypeVolume,
				Text:       text[start:end],
				Start:      start,
				End:        end,
				Confidence: 1.0,
				Value:      value,
				Unit:       unit,
			})
		}
	}
	
	return entities
}

// extractPowers извлекает мощности
func (ner *RussianNER) extractPowers(text string) []NEREntity {
	var entities []NEREntity
	matches := ner.powerRegex.FindAllStringSubmatchIndex(text, -1)
	
	for _, match := range matches {
		if len(match) >= 4 {
			start, end := match[0], match[1]
			value := text[match[2]:match[3]]
			unit := text[match[4]:match[5]]
			
			entities = append(entities, NEREntity{
				Type:       NEREntityTypePower,
				Text:       text[start:end],
				Start:      start,
				End:        end,
				Confidence: 1.0,
				Value:      value,
				Unit:       unit,
			})
		}
	}
	
	return entities
}

// extractCodes извлекает коды
func (ner *RussianNER) extractCodes(text string) []NEREntity {
	var entities []NEREntity
	// Используем FindAllString для поиска всех совпадений
	matches := ner.codeRegex.FindAllString(text, -1)
	matchIndices := ner.codeRegex.FindAllStringIndex(text, -1)
	
	for i, code := range matches {
		if i < len(matchIndices) {
			start, end := matchIndices[i][0], matchIndices[i][1]
			entities = append(entities, NEREntity{
				Type:       NEREntityTypeCode,
				Text:       code,
				Start:      start,
				End:        end,
				Confidence: 1.0,
				Value:      code,
			})
		}
	}
	
	return entities
}

// removeOverlapping удаляет перекрывающиеся сущности
func (ner *RussianNER) removeOverlapping(entities []NEREntity) []NEREntity {
	if len(entities) == 0 {
		return entities
	}
	
	// Сортируем по позиции начала
	// (в реальной реализации можно использовать более сложную логику)
	
	// Простая реализация: оставляем сущности с большей уверенностью
	result := make([]NEREntity, 0, len(entities))
	used := make(map[int]bool)
	
	for i, entity1 := range entities {
		if used[i] {
			continue
		}
		
		overlaps := false
		for j, entity2 := range entities {
			if i == j || used[j] {
				continue
			}
			
			// Проверяем перекрытие
			if (entity1.Start < entity2.End && entity1.End > entity2.Start) {
				// Перекрываются - выбираем с большей уверенностью
				if entity2.Confidence > entity1.Confidence {
					overlaps = true
					break
				}
			}
		}
		
		if !overlaps {
			result = append(result, entity1)
			used[i] = true
		}
	}
	
	return result
}

// TagWithBIO выполняет BIO-тегирование токенов
func (ner *RussianNER) TagWithBIO(text string) []BIOTaggedToken {
	entities := ner.ExtractEntities(text)
	words := strings.Fields(text)
	
	tokens := make([]BIOTaggedToken, 0, len(words))
	pos := 0
	
	for _, word := range words {
		wordStart := strings.Index(text[pos:], word)
		if wordStart < 0 {
			continue
		}
		wordStart += pos
		wordEnd := wordStart + len(word)
		
		// Ищем сущность для этого слова
		tag := BIO_O
		entityType := NEREntityType("")
		
		for _, entity := range entities {
			if wordStart >= entity.Start && wordEnd <= entity.End {
				if wordStart == entity.Start {
					tag = BIO_B
				} else {
					tag = BIO_I
				}
				entityType = entity.Type
				break
			}
		}
		
		tokens = append(tokens, BIOTaggedToken{
			Token:     word,
			Tag:       tag,
			EntityType: entityType,
			Start:     wordStart,
			End:       wordEnd,
		})
		
		pos = wordEnd
	}
	
	return tokens
}

// AddMaterial добавляет материал в словарь
func (ner *RussianNER) AddMaterial(word, normalized string) {
	ner.materials[strings.ToLower(word)] = normalized
}

// AddColor добавляет цвет в словарь
func (ner *RussianNER) AddColor(word, normalized string) {
	ner.colors[strings.ToLower(word)] = normalized
}

// AddType добавляет тип в словарь
func (ner *RussianNER) AddType(word, normalized string) {
	ner.types[strings.ToLower(word)] = normalized
}

