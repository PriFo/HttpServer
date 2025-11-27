package algorithms

import (
	"strings"
	"unicode"
)

// SoundexRU реализует алгоритм Soundex для русского языка
// Soundex кодирует слова в код, где похожие по звучанию слова получают одинаковый код
type SoundexRU struct{}

// NewSoundexRU создает новый экземпляр SoundexRU
func NewSoundexRU() *SoundexRU {
	return &SoundexRU{}
}

// Encode кодирует строку в Soundex код
// Алгоритм:
// 1. Первая буква сохраняется
// 2. Остальные буквы кодируются цифрами по правилам
// 3. Удаляются повторяющиеся цифры
// 4. Результат обрезается до 4 символов
func (s *SoundexRU) Encode(text string) string {
	if text == "" {
		return ""
	}

	// Приводим к верхнему регистру и удаляем пробелы
	text = strings.ToUpper(strings.TrimSpace(text))
	if text == "" {
		return ""
	}

	// Удаляем все символы, кроме букв
	var cleaned strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) && (r >= 'А' && r <= 'Я' || r == 'Ё') {
			cleaned.WriteRune(r)
		}
	}
	text = cleaned.String()
	if text == "" {
		return ""
	}

	// Первая буква (используем руны для корректной работы с кириллицей)
	textRunes := []rune(text)
	if len(textRunes) == 0 {
		return ""
	}
	firstChar := string(textRunes[0])
	result := strings.Builder{}
	result.WriteString(firstChar)

	// Кодируем остальные буквы (максимум 3 цифры для 4-символьного кода)
	prevCode := ""
	digitCount := 0
	maxDigits := 3
	for i := 1; i < len([]rune(text)) && digitCount < maxDigits; i++ {
		char := string([]rune(text)[i])
		code := s.getCode(char)

		// Пропускаем, если код такой же как у предыдущего
		if code != "" && code != prevCode {
			result.WriteString(code)
			prevCode = code
			digitCount++
		}
	}

	// Обрезаем до 4 символов и дополняем нулями
	encoded := result.String()
	encodedRunes := []rune(encoded)
	if len(encodedRunes) > 4 {
		encoded = string(encodedRunes[:4])
	} else {
		encoded = encoded + strings.Repeat("0", 4-len(encodedRunes))
	}

	return encoded
}

// getCode возвращает код для буквы согласно правилам Soundex для русского языка
// Правила основаны на фонетическом сходстве звуков:
// 1 - Б, П, Ф, В (губные)
// 2 - Г, К, Х (заднеязычные)
// 3 - Д, Т (переднеязычные)
// 4 - Л (латеральный)
// 5 - М, Н (носовые)
// 6 - Р (дрожащий)
// 7 - Ж, Ш, Щ, Ч, Ц (шипящие и аффрикаты)
// 8 - З, С (свистящие)
// 9 - Й (полугласный)
func (s *SoundexRU) getCode(char string) string {
	// Группа 1: Б, П, Ф, В
	if strings.ContainsAny(char, "БПФВ") {
		return "1"
	}
	// Группа 2: Г, К, Х
	if strings.ContainsAny(char, "ГКХ") {
		return "2"
	}
	// Группа 3: Д, Т
	if strings.ContainsAny(char, "ДТ") {
		return "3"
	}
	// Группа 4: Л
	if char == "Л" {
		return "4"
	}
	// Группа 5: М, Н
	if strings.ContainsAny(char, "МН") {
		return "5"
	}
	// Группа 6: Р
	if char == "Р" {
		return "6"
	}
	// Группа 7: Ж, Ш, Щ, Ч, Ц
	if strings.ContainsAny(char, "ЖШЩЧЦ") {
		return "7"
	}
	// Группа 8: З, С
	if strings.ContainsAny(char, "ЗС") {
		return "8"
	}
	// Группа 9: Й
	if char == "Й" {
		return "9"
	}
	// Гласные и другие буквы не кодируются
	return ""
}

// Compare сравнивает две строки используя Soundex
// Возвращает true, если коды совпадают
func (s *SoundexRU) Compare(str1, str2 string) bool {
	code1 := s.Encode(str1)
	code2 := s.Encode(str2)
	return code1 != "" && code1 == code2
}

// Similarity возвращает схожесть двух строк на основе Soundex
// Возвращает 1.0 если коды совпадают, 0.0 если не совпадают
func (s *SoundexRU) Similarity(str1, str2 string) float64 {
	code1 := s.Encode(str1)
	code2 := s.Encode(str2)

	if code1 == "" || code2 == "" {
		return 0.0
	}

	if code1 == code2 {
		return 1.0
	}

	// Частичное совпадение: считаем совпадающие позиции
	matches := 0
	minLen := len(code1)
	if len(code2) < minLen {
		minLen = len(code2)
	}

	for i := 0; i < minLen; i++ {
		if code1[i] == code2[i] {
			matches++
		}
	}

	// Возвращаем долю совпадений
	return float64(matches) / float64(4) // 4 - длина Soundex кода
}

