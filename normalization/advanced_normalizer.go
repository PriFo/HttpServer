package normalization

import (
	"regexp"
	"strings"
	"unicode"
)

// AdvancedNormalizer предоставляет расширенные методы нормализации наименований НСИ
type AdvancedNormalizer struct {
	// Таблица транслитерации кириллица -> латиница
	transliterationMap map[rune]string
	// Таблица транслитерации латиница -> кириллица (обратная)
	reverseTransliterationMap map[string]rune
	// Словарь аббревиатур и их расшифровок
	abbreviations map[string]string
	// Словарь единиц измерения для нормализации
	unitNormalization map[string]string
}

// NewAdvancedNormalizer создает новый расширенный нормализатор
func NewAdvancedNormalizer() *AdvancedNormalizer {
	an := &AdvancedNormalizer{
		transliterationMap:        make(map[rune]string),
		reverseTransliterationMap: make(map[string]rune),
		abbreviations:             make(map[string]string),
		unitNormalization:         make(map[string]string),
	}

	// Инициализация таблицы транслитерации (ГОСТ 7.79-2000)
	an.initTransliteration()
	an.initAbbreviations()
	an.initUnitNormalization()

	return an
}

// initTransliteration инициализирует таблицу транслитерации
func (an *AdvancedNormalizer) initTransliteration() {
	transMap := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
		'е': "e", 'ё': "yo", 'ж': "zh", 'з': "z", 'и': "i",
		'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
		'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
		'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch",
		'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "",
		'э': "e", 'ю': "yu", 'я': "ya",
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D",
		'Е': "E", 'Ё': "Yo", 'Ж': "Zh", 'З': "Z", 'И': "I",
		'Й': "Y", 'К': "K", 'Л': "L", 'М': "M", 'Н': "N",
		'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T",
		'У': "U", 'Ф': "F", 'Х': "Kh", 'Ц': "Ts", 'Ч': "Ch",
		'Ш': "Sh", 'Щ': "Shch", 'Ъ': "", 'Ы': "Y", 'Ь': "",
		'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	}

	for k, v := range transMap {
		an.transliterationMap[k] = v
		if v != "" {
			an.reverseTransliterationMap[strings.ToLower(v)] = unicode.ToLower(k)
		}
	}
}

// initAbbreviations инициализирует словарь аббревиатур
func (an *AdvancedNormalizer) initAbbreviations() {
	an.abbreviations = map[string]string{
		"мм":  "миллиметр",
		"см":  "сантиметр",
		"м":   "метр",
		"кг":  "килограмм",
		"г":   "грамм",
		"л":   "литр",
		"шт":  "штука",
		"м2":  "квадратный метр",
		"м3":  "кубический метр",
		"вт":  "ватт",
		"квт": "киловатт",
		"а":   "ампер",
		"в":   "вольт",
		"ооо": "общество с ограниченной ответственностью",
		"оао": "открытое акционерное общество",
		"зао": "закрытое акционерное общество",
		"ип":  "индивидуальный предприниматель",
		"пб":  "производство",
		"пбю": "производственно-бытовое управление",
		"тд":  "торговый дом",
		"тк":  "торговый комплекс",
	}
}

// initUnitNormalization инициализирует словарь нормализации единиц измерения
func (an *AdvancedNormalizer) initUnitNormalization() {
	an.unitNormalization = map[string]string{
		// Длина
		"мм": "мм", "mm": "мм", "миллиметр": "мм", "millimeter": "мм",
		"см": "см", "cm": "см", "сантиметр": "см", "centimeter": "см",
		"м": "м", "m": "м", "метр": "м", "meter": "м", "metre": "м",
		"км": "км", "km": "км", "километр": "км", "kilometer": "км",
		// Масса
		"г": "г", "g": "г", "грамм": "г", "gram": "г",
		"кг": "кг", "kg": "кг", "килограмм": "кг", "kilogram": "кг",
		"т": "т", "t": "т", "тонна": "т", "ton": "т", "tonne": "т",
		// Объем
		"л": "л", "l": "л", "литр": "л", "liter": "л", "litre": "л",
		"мл": "мл", "ml": "мл", "миллилитр": "мл", "milliliter": "мл",
		// Мощность
		"вт": "вт", "w": "вт", "watt": "вт", "ватт": "вт",
		"квт": "квт", "kw": "квт", "kilowatt": "квт", "киловатт": "квт",
		// Электричество
		"а": "а", "a": "а", "ампер": "а", "ampere": "а",
		"в": "в", "v": "в", "вольт": "в", "volt": "в",
		// Время
		"сек": "сек", "sec": "сек", "секунда": "сек", "second": "сек",
		"мин": "мин", "min": "мин", "минута": "мин", "minute": "мин",
		"ч": "ч", "h": "ч", "час": "ч", "hour": "ч",
		// Количество
		"шт": "шт", "pcs": "шт", "штука": "шт", "piece": "шт",
	}
}

// Transliterate транслитерирует текст из кириллицы в латиницу
func (an *AdvancedNormalizer) Transliterate(text string) string {
	var result strings.Builder
	for _, r := range text {
		if replacement, ok := an.transliterationMap[r]; ok {
			result.WriteString(replacement)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ReverseTransliterate транслитерирует текст из латиницы в кириллицу
func (an *AdvancedNormalizer) ReverseTransliterate(text string) string {
	text = strings.ToLower(text)
	var result strings.Builder
	i := 0
	for i < len(text) {
		matched := false
		// Пробуем найти совпадение для последовательностей символов (для shch, sh, ch и т.д.)
		for length := 4; length >= 1 && i+length <= len(text); length-- {
			substr := text[i : i+length]
			if r, ok := an.reverseTransliterationMap[substr]; ok {
				result.WriteRune(r)
				i += length
				matched = true
				break
			}
		}
		if !matched {
			result.WriteByte(text[i])
			i++
		}
	}
	return result.String()
}

// RemoveDiacritics удаляет диакритические знаки из текста
func (an *AdvancedNormalizer) RemoveDiacritics(text string) string {
	var result strings.Builder
	for _, r := range text {
		// Приводим к базовой форме (удаляем диакритику)
		switch r {
		case 'ё', 'Ё':
			result.WriteRune('е')
		case 'й', 'Й':
			result.WriteRune('и')
		default:
			// Для других символов просто оставляем как есть
			// (удаление диакритики для не-кириллицы не требуется)
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Stem выполняет простой стемминг для русского языка
// Удаляет стандартные окончания для приведения к базовой форме
func (an *AdvancedNormalizer) Stem(word string) string {
	word = strings.ToLower(word)
	if len(word) < 4 {
		return word
	}

	// Список окончаний для удаления (от длинных к коротким)
	endings := []string{
		"ами", "ами", "ами", "ами", "ами", // множественное число, творительный падеж
		"ами", "ами", "ами", "ами", "ами",
		"ов", "ев", "ей", "ий", "ый", // множественное число, родительный падеж
		"ах", "ях", "ях", "ях", "ях", // множественное число, предложный падеж
		"ом", "ем", "ем", "ем", "ем", // единственное число, творительный падеж
		"ой", "ей", "ей", "ей", "ей", // единственное число, творительный падеж
		"ах", "ях", "ях", "ях", "ях", // единственное число, предложный падеж
		"ов", "ев", "ей", "ий", "ый", // прилагательные
		"ая", "яя", "яя", "яя", "яя", // прилагательные, женский род
		"ое", "ее", "ее", "ее", "ее", // прилагательные, средний род
		"ые", "ие", "ие", "ие", "ие", // прилагательные, множественное число
		"ый", "ий", "ий", "ий", "ий", // прилагательные, мужской род
		"а", "я", "я", "я", "я", // единственное число, именительный падеж
		"о", "е", "е", "е", "е", // единственное число, средний род
		"у", "ю", "ю", "ю", "ю", // единственное число, дательный падеж
		"и", "и", "и", "и", "и", // множественное число, именительный падеж
		"ы", "и", "и", "и", "и", // множественное число, именительный падеж
		"ь", "ь", "ь", "ь", "ь", // мягкий знак
	}

	for _, ending := range endings {
		if strings.HasSuffix(word, ending) && len(word) > len(ending)+2 {
			return word[:len(word)-len(ending)]
		}
	}

	return word
}

// StemText выполняет стемминг для всего текста
func (an *AdvancedNormalizer) StemText(text string) string {
	words := strings.Fields(text)
	var stemmedWords []string
	for _, word := range words {
		// Удаляем знаки препинания
		word = strings.Trim(word, ".,!?;:()[]{}")
		if word != "" {
			stemmedWords = append(stemmedWords, an.Stem(word))
		}
	}
	return strings.Join(stemmedWords, " ")
}

// NormalizeUnits нормализует единицы измерения в тексте
func (an *AdvancedNormalizer) NormalizeUnits(text string) string {
	// Ищем единицы измерения и заменяем их на стандартные
	words := strings.Fields(text)
	var normalizedWords []string
	for _, word := range words {
		wordLower := strings.ToLower(word)
		// Удаляем знаки препинания для проверки
		wordClean := strings.Trim(wordLower, ".,!?;:()[]{}")
		if normalized, ok := an.unitNormalization[wordClean]; ok {
			normalizedWords = append(normalizedWords, normalized)
		} else {
			normalizedWords = append(normalizedWords, word)
		}
	}
	return strings.Join(normalizedWords, " ")
}

// ExpandAbbreviations расшифровывает аббревиатуры в тексте
func (an *AdvancedNormalizer) ExpandAbbreviations(text string) string {
	words := strings.Fields(text)
	var expandedWords []string
	for _, word := range words {
		wordLower := strings.ToLower(strings.Trim(word, ".,!?;:()[]{}"))
		if expanded, ok := an.abbreviations[wordLower]; ok {
			expandedWords = append(expandedWords, expanded)
		} else {
			expandedWords = append(expandedWords, word)
		}
	}
	return strings.Join(expandedWords, " ")
}

// NormalizeWhitespace нормализует пробелы и регистр
func (an *AdvancedNormalizer) NormalizeWhitespace(text string) string {
	// Удаляем множественные пробелы
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	// Удаляем пробелы в начале и конце
	text = strings.TrimSpace(text)
	// Приводим к нижнему регистру
	text = strings.ToLower(text)
	return text
}

// AdvancedNormalize выполняет комплексную нормализацию с использованием всех методов
func (an *AdvancedNormalizer) AdvancedNormalize(text string, options NormalizationOptions) string {
	if text == "" {
		return ""
	}

	result := text

	// 1. Нормализация пробелов и регистра
	if options.NormalizeWhitespace {
		result = an.NormalizeWhitespace(result)
	}

	// 2. Удаление диакритических знаков
	if options.RemoveDiacritics {
		result = an.RemoveDiacritics(result)
	}

	// 3. Транслитерация (если нужно)
	if options.Transliterate {
		result = an.Transliterate(result)
	}

	// 4. Расшифровка аббревиатур
	if options.ExpandAbbreviations {
		result = an.ExpandAbbreviations(result)
	}

	// 5. Нормализация единиц измерения
	if options.NormalizeUnits {
		result = an.NormalizeUnits(result)
	}

	// 6. Стемминг
	if options.Stem {
		result = an.StemText(result)
	}

	// Финальная нормализация пробелов
	result = strings.TrimSpace(result)

	return result
}

// NormalizationOptions опции для расширенной нормализации
type NormalizationOptions struct {
	NormalizeWhitespace bool // Нормализация пробелов и регистра
	RemoveDiacritics    bool // Удаление диакритических знаков
	Transliterate       bool // Транслитерация
	ExpandAbbreviations bool // Расшифровка аббревиатур
	NormalizeUnits      bool // Нормализация единиц измерения
	Stem                bool // Стемминг
}

// DefaultNormalizationOptions возвращает опции по умолчанию
func DefaultNormalizationOptions() NormalizationOptions {
	return NormalizationOptions{
		NormalizeWhitespace: true,
		RemoveDiacritics:    false,
		Transliterate:       false,
		ExpandAbbreviations: false,
		NormalizeUnits:      true,
		Stem:                false,
	}
}
