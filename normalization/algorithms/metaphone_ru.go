package algorithms

import (
	"strings"
	"unicode"
)

// MetaphoneRU реализует улучшенный фонетический алгоритм Metaphone для русского языка
// Metaphone более точный чем Soundex, учитывает контекст и позицию звуков
type MetaphoneRU struct{}

// NewMetaphoneRU создает новый экземпляр MetaphoneRU
func NewMetaphoneRU() *MetaphoneRU {
	return &MetaphoneRU{}
}

// Encode кодирует строку в Metaphone код
// Алгоритм более сложный чем Soundex, учитывает контекст букв
func (m *MetaphoneRU) Encode(text string) string {
	if text == "" {
		return ""
	}

	// Приводим к верхнему регистру
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

	// Преобразуем в руны для работы с Unicode
	runes := []rune(text)
	length := len(runes)

	// Первая буква
	result := strings.Builder{}
	result.WriteRune(runes[0])

	// Обрабатываем остальные буквы с учетом контекста
	i := 1
	for i < length {
		prev := ""
		curr := string(runes[i])
		next := ""
		nextNext := ""

		if i > 0 {
			prev = string(runes[i-1])
		}
		if i < length-1 {
			next = string(runes[i+1])
		}
		if i < length-2 {
			nextNext = string(runes[i+2])
		}

		code := m.getCode(curr, prev, next, nextNext, i == length-1)
		if code != "" {
			result.WriteString(code)
		}

		// Пропускаем обработанные буквы в диграфах
		if m.isDigraph(curr, next) {
			i += 2
		} else {
			i++
		}
	}

	// Ограничиваем длину результата (обычно 4-6 символов)
	encoded := result.String()
	if len(encoded) > 6 {
		encoded = encoded[:6]
	}

	return encoded
}

// getCode возвращает код для буквы с учетом контекста
func (m *MetaphoneRU) getCode(curr, prev, next, nextNext string, isLast bool) string {
	// Обработка диграфов (двухбуквенных сочетаний)
	if m.isDigraph(curr, next) {
		return m.getDigraphCode(curr, next)
	}

	// Правила для отдельных букв с учетом контекста

	// Б, П - кодируются как 1, но П в начале слова или после гласной
	if curr == "Б" {
		return "1"
	}
	if curr == "П" {
		// П в начале или после гласной
		if prev == "" || m.isVowel(prev) {
			return "1"
		}
		// П перед глухими согласными
		if next != "" && m.isVoiceless(next) {
			return "1"
		}
		return "1"
	}

	// В, Ф - кодируются как 2
	if strings.ContainsAny(curr, "ВФ") {
		return "2"
	}

	// Г, К, Х - кодируются как 3
	if strings.ContainsAny(curr, "ГКХ") {
		return "3"
	}

	// Д, Т - кодируются как 4
	if strings.ContainsAny(curr, "ДТ") {
		return "4"
	}

	// Л - кодируется как 5
	if curr == "Л" {
		return "5"
	}

	// М, Н - кодируются как 6
	if strings.ContainsAny(curr, "МН") {
		return "6"
	}

	// Р - кодируется как 7
	if curr == "Р" {
		return "7"
	}

	// Ж, Ш, Щ - кодируются как 8
	if strings.ContainsAny(curr, "ЖШЩ") {
		return "8"
	}

	// Ч, Ц - кодируются как 9
	if strings.ContainsAny(curr, "ЧЦ") {
		return "9"
	}

	// З, С - кодируются как 0
	if strings.ContainsAny(curr, "ЗС") {
		return "0"
	}

	// Й - кодируется как J
	if curr == "Й" {
		return "J"
	}

	// Гласные не кодируются (пропускаются)
	if m.isVowel(curr) {
		return ""
	}

	return ""
}

// isDigraph проверяет, является ли сочетание диграфом
func (m *MetaphoneRU) isDigraph(curr, next string) bool {
	if next == "" {
		return false
	}

	// Русские диграфы: ЖЧ, ШЧ, ЧШ и т.д.
	digraphs := []string{"ЖЧ", "ШЧ", "ЧШ", "ЩЧ", "ЧЩ"}
	combined := curr + next
	for _, d := range digraphs {
		if combined == d {
			return true
		}
	}
	return false
}

// getDigraphCode возвращает код для диграфа
func (m *MetaphoneRU) getDigraphCode(curr, next string) string {
	combined := curr + next
	switch combined {
	case "ЖЧ", "ШЧ", "ЧШ", "ЩЧ", "ЧЩ":
		return "8" // Шипящие диграфы
	default:
		return ""
	}
}

// isVowel проверяет, является ли буква гласной
func (m *MetaphoneRU) isVowel(char string) bool {
	vowels := "АЕЁИОУЫЭЮЯ"
	return strings.ContainsAny(char, vowels)
}

// isVoiceless проверяет, является ли согласная глухой
func (m *MetaphoneRU) isVoiceless(char string) bool {
	voiceless := "ПТКФСШЧЦХ"
	return strings.ContainsAny(char, voiceless)
}

// Compare сравнивает две строки используя Metaphone
// Возвращает true, если коды совпадают
func (m *MetaphoneRU) Compare(str1, str2 string) bool {
	code1 := m.Encode(str1)
	code2 := m.Encode(str2)
	return code1 != "" && code1 == code2
}

// Similarity возвращает схожесть двух строк на основе Metaphone
// Возвращает 1.0 если коды совпадают, иначе частичное совпадение
func (m *MetaphoneRU) Similarity(str1, str2 string) float64 {
	code1 := m.Encode(str1)
	code2 := m.Encode(str2)

	if code1 == "" || code2 == "" {
		return 0.0
	}

	if code1 == code2 {
		return 1.0
	}

	// Вычисляем расстояние Левенштейна между кодами
	maxLen := len(code1)
	if len(code2) > maxLen {
		maxLen = len(code2)
	}

	if maxLen == 0 {
		return 1.0
	}

	// Простое сравнение по позициям
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

	// Возвращаем долю совпадений с учетом длины
	return float64(matches) / float64(maxLen)
}

