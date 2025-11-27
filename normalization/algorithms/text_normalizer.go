package algorithms

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// TextNormalizer нормализует текст для дальнейшей обработки
type TextNormalizer struct {
	removeStopWords bool
	stopWords       map[string]bool
}

// NewTextNormalizer создает новый нормализатор текста
func NewTextNormalizer(removeStopWords bool) *TextNormalizer {
	normalizer := &TextNormalizer{
		removeStopWords: removeStopWords,
		stopWords:       getDefaultStopWordsForNormalizer(),
	}
	return normalizer
}

// Normalize выполняет полную нормализацию текста
func (tn *TextNormalizer) Normalize(text string) string {
	// 1. Приведение к нижнему регистру
	text = strings.ToLower(text)

	// 2. Удаление лишних пробелов
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")

	// 3. Нормализация кавычек
	text = normalizeQuotes(text)

	// 4. Нормализация дефисов
	text = normalizeHyphens(text)

	// 5. Нормализация Unicode (приведение к NFC форме)
	text = normalizeUnicode(text)

	// 6. Удаление диакритических знаков (для латиницы)
	text = removeDiacritics(text)

	// 7. Удаление стоп-слов (если включено)
	if tn.removeStopWords {
		text = tn.removeStopWordsFromText(text)
	}

	return strings.TrimSpace(text)
}

// normalizeQuotes нормализует различные типы кавычек
func normalizeQuotes(text string) string {
	replacements := map[rune]rune{
		'\u201C': '"', // Left double quotation mark
		'\u201D': '"', // Right double quotation mark
		'\u2018': '\'', // Left single quotation mark
		'\u2019': '\'', // Right single quotation mark
		'«':      '"',  // French quotes
		'»':      '"',
		'„':      '"',  // German low double quotes
		'‚':      '\'', // Single low quote
	}

	var builder strings.Builder
	for _, r := range text {
		if replacement, ok := replacements[r]; ok {
			builder.WriteRune(replacement)
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// normalizeHyphens нормализует различные типы дефисов
func normalizeHyphens(text string) string {
	// Заменяем длинные тире и другие варианты на обычный дефис
	text = strings.ReplaceAll(text, "—", "-")
	text = strings.ReplaceAll(text, "–", "-")
	text = strings.ReplaceAll(text, "−", "-")
	return text
}

// removeDiacritics удаляет диакритические знаки (для латиницы)
func removeDiacritics(text string) string {
	replacements := map[rune]rune{
		'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a',
		'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
		'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
		'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
		'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
		'ý': 'y', 'ÿ': 'y',
		'ñ': 'n', 'ç': 'c',
		'Á': 'A', 'À': 'A', 'Â': 'A', 'Ã': 'A', 'Ä': 'A', 'Å': 'A',
		'É': 'E', 'È': 'E', 'Ê': 'E', 'Ë': 'E',
		'Í': 'I', 'Ì': 'I', 'Î': 'I', 'Ï': 'I',
		'Ó': 'O', 'Ò': 'O', 'Ô': 'O', 'Õ': 'O', 'Ö': 'O',
		'Ú': 'U', 'Ù': 'U', 'Û': 'U', 'Ü': 'U',
		'Ý': 'Y',
		'Ñ': 'N', 'Ç': 'C',
	}

	var builder strings.Builder
	for _, r := range text {
		if replacement, ok := replacements[r]; ok {
			builder.WriteRune(replacement)
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// removeStopWordsFromText удаляет стоп-слова из текста
func (tn *TextNormalizer) removeStopWordsFromText(text string) string {
	words := strings.Fields(text)
	result := make([]string, 0, len(words))

	for _, word := range words {
		word = strings.TrimSpace(word)
		if word != "" && !tn.stopWords[word] {
			result = append(result, word)
		}
	}

	return strings.Join(result, " ")
}

// getDefaultStopWordsForNormalizer возвращает список стоп-слов для нормализатора
// Использует общую функцию из token_based.go
func getDefaultStopWordsForNormalizer() map[string]bool {
	// Используем общую функцию, если она доступна, иначе возвращаем локальную версию
	return map[string]bool{
		// Русские стоп-слова
		"и": true, "в": true, "на": true, "с": true, "для": true,
		"по": true, "из": true, "к": true, "от": true, "о": true,
		"а": true, "но": true, "или": true, "то": true, "что": true,
		"как": true, "так": true, "это": true, "он": true, "она": true,
		"они": true, "мы": true, "вы": true, "его": true, "её": true,
		"их": true, "этот": true, "эта": true, "эти": true,
		
		// Английские стоп-слова
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "have": true, "has": true, "had": true,
		"this": true, "that": true, "these": true, "those": true,
	}
}

// Transliterate выполняет транслитерацию между кириллицей и латиницей
func Transliterate(text string, toLatin bool) string {
	if toLatin {
		return cyrillicToLatin(text)
	}
	return latinToCyrillic(text)
}

// cyrillicToLatin транслитерирует кириллицу в латиницу
func cyrillicToLatin(text string) string {
	translitMap := map[rune]string{
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

	var builder strings.Builder
	for _, r := range text {
		if translit, ok := translitMap[r]; ok {
			builder.WriteString(translit)
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// latinToCyrillic транслитерирует латиницу в кириллицу (упрощенная версия)
func latinToCyrillic(text string) string {
	// Это упрощенная версия, полная транслитерация требует более сложной логики
	translitMap := map[string]rune{
		"a": 'а', "b": 'б', "v": 'в', "g": 'г', "d": 'д',
		"e": 'е', "yo": 'ё', "zh": 'ж', "z": 'з', "i": 'и',
		"j": 'й', "k": 'к', "l": 'л', "m": 'м', "n": 'н',
		"o": 'о', "p": 'п', "r": 'р', "s": 'с', "t": 'т',
		"u": 'у', "f": 'ф', "kh": 'х', "ts": 'ц', "ch": 'ч',
		"sh": 'ш', "shch": 'щ', "y": 'ы', "eh": 'э',
		"yu": 'ю', "ya": 'я',
	}

	// Упрощенная реализация - ищем совпадения по порядку
	result := text
	for latin, cyrillic := range translitMap {
		result = strings.ReplaceAll(result, latin, string(cyrillic))
	}
	return result
}

// RemovePunctuation удаляет знаки пунктуации
func RemovePunctuation(text string) string {
	var builder strings.Builder
	for _, r := range text {
		if !unicode.IsPunct(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// RemoveNumbers удаляет числа из текста
func RemoveNumbers(text string) string {
	var builder strings.Builder
	for _, r := range text {
		if !unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// NormalizeWhitespace нормализует пробельные символы
func NormalizeWhitespace(text string) string {
	// Заменяем все пробельные символы на обычные пробелы
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	
	// Удаляем множественные пробелы
	words := strings.Fields(text)
	return strings.Join(words, " ")
}

// normalizeUnicode выполняет нормализацию Unicode
// Приводит текст к канонической форме (упрощенная версия без внешних зависимостей)
// Удаляет комбинирующие диакритические знаки и нормализует составные символы
func normalizeUnicode(text string) string {
	if text == "" {
		return text
	}

	var result strings.Builder
	result.Grow(len(text))

	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		text = text[size:]

		// Удаляем комбинирующие диакритические знаки (Combining Diacritical Marks)
		// Диапазон: U+0300-U+036F
		if r >= 0x0300 && r <= 0x036F {
			continue
		}

		// Удаляем другие комбинирующие знаки
		// Диапазон: U+1AB0-U+1AFF (Combining Diacritical Marks Extended)
		if r >= 0x1AB0 && r <= 0x1AFF {
			continue
		}

		// Удаляем комбинирующие знаки для символов (U+1DC0-U+1DFF)
		if r >= 0x1DC0 && r <= 0x1DFF {
			continue
		}

		// Удаляем комбинирующие знаки для символов (U+20D0-U+20FF)
		if r >= 0x20D0 && r <= 0x20FF {
			continue
		}

		// Нормализуем составные символы (например, é может быть представлен как e+́)
		// Это упрощенная версия - полная нормализация требует golang.org/x/text/unicode/norm
		// Но для большинства случаев достаточно удаления комбинирующих знаков

		result.WriteRune(r)
	}

	return result.String()
}

