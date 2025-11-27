package algorithms

import (
	"strings"
	"unicode"
)

// Soundex реализует алгоритм Soundex для русского языка
type Soundex struct{}

// NewSoundex создает новый экземпляр Soundex
func NewSoundex() *Soundex {
	return &Soundex{}
}

// Encode кодирует строку в Soundex код
// Soundex код состоит из первой буквы и трех цифр
func (s *Soundex) Encode(text string) string {
	if text == "" {
		return ""
	}

	text = strings.ToUpper(strings.TrimSpace(text))
	if text == "" {
		return ""
	}

	// Маппинг русских букв на цифры
	// 0: гласные и мягкие знаки (игнорируются)
	// 1: б, п, ф, в
	// 2: г, к, х
	// 3: д, т
	// 4: ж, ш, щ, ч
	// 5: з, с, ц
	// 6: л
	// 7: м, н
	// 8: р
	// 9: й (игнорируется в начале)
	mapping := map[rune]int{
		'А': 0, 'Е': 0, 'Ё': 0, 'И': 0, 'О': 0, 'У': 0, 'Ы': 0, 'Э': 0, 'Ю': 0, 'Я': 0,
		'Б': 1, 'П': 1, 'Ф': 1, 'В': 1,
		'Г': 2, 'К': 2, 'Х': 2,
		'Д': 3, 'Т': 3,
		'Ж': 4, 'Ш': 4, 'Щ': 4, 'Ч': 4,
		'З': 5, 'С': 5, 'Ц': 5,
		'Л': 6,
		'М': 7, 'Н': 7,
		'Р': 8,
		'Й': 9,
		'Ь': 0, 'Ъ': 0,
	}

	// Первая буква
	firstChar := rune(text[0])
	code := strings.Builder{}
	code.WriteRune(firstChar)

	lastCode := -1
	if codeNum, ok := mapping[firstChar]; ok {
		lastCode = codeNum
	}

	// Обрабатываем остальные символы
	for _, char := range text[1:] {
		if !unicode.IsLetter(char) {
			continue
		}

		codeNum, ok := mapping[char]
		if !ok {
			// Для английских букв используем стандартный Soundex
			codeNum = s.englishSoundexCode(char)
		}

		// Пропускаем гласные и дубликаты
		if codeNum != 0 && codeNum != lastCode {
			code.WriteRune(rune('0' + codeNum))
			lastCode = codeNum
			if code.Len() >= 4 {
				break
			}
		}
	}

	// Дополняем нулями до 4 символов
	result := code.String()
	for len(result) < 4 {
		result += "0"
	}

	return result[:4]
}

// englishSoundexCode возвращает код для английской буквы
func (s *Soundex) englishSoundexCode(char rune) int {
	char = unicode.ToUpper(char)
	switch char {
	case 'B', 'F', 'P', 'V':
		return 1
	case 'C', 'G', 'J', 'K', 'Q', 'S', 'X', 'Z':
		return 2
	case 'D', 'T':
		return 3
	case 'L':
		return 4
	case 'M', 'N':
		return 5
	case 'R':
		return 6
	default:
		return 0
	}
}

// Similarity вычисляет схожесть двух строк на основе Soundex
func (s *Soundex) Similarity(text1, text2 string) float64 {
	code1 := s.Encode(text1)
	code2 := s.Encode(text2)

	if code1 == code2 {
		return 1.0
	}

	// Считаем количество совпадающих позиций
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

	return float64(matches) / float64(minLen)
}

// Metaphone реализует алгоритм Metaphone для русского языка
type Metaphone struct{}

// NewMetaphone создает новый экземпляр Metaphone
func NewMetaphone() *Metaphone {
	return &Metaphone{}
}

// Encode кодирует строку в Metaphone код
// Metaphone более сложный алгоритм, который учитывает контекст
func (m *Metaphone) Encode(text string) string {
	if text == "" {
		return ""
	}

	text = strings.ToUpper(strings.TrimSpace(text))
	if text == "" {
		return ""
	}

	// Упрощенная версия Metaphone для русского языка
	// Удаляем гласные в конце (кроме первой буквы)
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteRune(runes[0])

	vowels := map[rune]bool{
		'А': true, 'Е': true, 'Ё': true, 'И': true, 'О': true,
		'У': true, 'Ы': true, 'Э': true, 'Ю': true, 'Я': true,
	}

	// Применяем правила замены
	for i := 1; i < len(runes); i++ {
		char := runes[i]
		prevChar := runes[i-1]

		// Пропускаем гласные (кроме первой)
		if vowels[char] {
			continue
		}

		// Правила замены для похожих звуков
		switch char {
		case 'Б', 'П':
			if prevChar != 'Б' && prevChar != 'П' {
				result.WriteRune('П')
			}
		case 'В', 'Ф':
			if prevChar != 'В' && prevChar != 'Ф' {
				result.WriteRune('Ф')
			}
		case 'Г', 'К', 'Х':
			if prevChar != 'Г' && prevChar != 'К' && prevChar != 'Х' {
				result.WriteRune('К')
			}
		case 'Д', 'Т':
			if prevChar != 'Д' && prevChar != 'Т' {
				result.WriteRune('Т')
			}
		case 'Ж', 'Ш', 'Щ', 'Ч':
			if prevChar != 'Ж' && prevChar != 'Ш' && prevChar != 'Щ' && prevChar != 'Ч' {
				result.WriteRune('Ш')
			}
		case 'З', 'С', 'Ц':
			if prevChar != 'З' && prevChar != 'С' && prevChar != 'Ц' {
				result.WriteRune('С')
			}
		default:
			if unicode.IsLetter(char) {
				result.WriteRune(char)
			}
		}
	}

	code := result.String()
	// Ограничиваем длину кода
	if len(code) > 6 {
		return code[:6]
	}
	return code
}

// Similarity вычисляет схожесть двух строк на основе Metaphone
func (m *Metaphone) Similarity(text1, text2 string) float64 {
	code1 := m.Encode(text1)
	code2 := m.Encode(text2)

	if code1 == code2 {
		return 1.0
	}

	// Используем расстояние Левенштейна для сравнения кодов
	distance := levenshteinDistance(code1, code2)
	maxLen := len(code1)
	if len(code2) > maxLen {
		maxLen = len(code2)
	}

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// PhoneticMatcher комбинирует несколько фонетических алгоритмов
type PhoneticMatcher struct {
	soundex  *Soundex
	metaphone *Metaphone
}

// NewPhoneticMatcher создает новый фонетический сопоставитель
func NewPhoneticMatcher() *PhoneticMatcher {
	return &PhoneticMatcher{
		soundex:   NewSoundex(),
		metaphone: NewMetaphone(),
	}
}

// Similarity вычисляет комбинированную схожесть
// Использует среднее арифметическое Soundex и Metaphone
func (pm *PhoneticMatcher) Similarity(text1, text2 string) float64 {
	soundexSim := pm.soundex.Similarity(text1, text2)
	metaphoneSim := pm.metaphone.Similarity(text1, text2)

	return (soundexSim + metaphoneSim) / 2.0
}

// EncodeSoundex возвращает Soundex код
func (pm *PhoneticMatcher) EncodeSoundex(text string) string {
	return pm.soundex.Encode(text)
}

// EncodeMetaphone возвращает Metaphone код
func (pm *PhoneticMatcher) EncodeMetaphone(text string) string {
	return pm.metaphone.Encode(text)
}

// levenshteinDistance вычисляет расстояние Левенштейна между двумя строками
func levenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)
	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Используем оптимизированный алгоритм с одним массивом
	column := make([]int, len1+1)
	for i := 1; i <= len1; i++ {
		column[i] = i
	}

	for x := 1; x <= len2; x++ {
		column[0] = x
		lastDiag := x - 1
		for y := 1; y <= len1; y++ {
			oldDiag := column[y]
			cost := 0
			if r1[y-1] != r2[x-1] {
				cost = 1
			}
			column[y] = min3Phonetic(column[y]+1, column[y-1]+1, lastDiag+cost)
			lastDiag = oldDiag
		}
	}

	return column[len1]
}

// min3Phonetic возвращает минимальное из трех чисел
func min3Phonetic(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// Функции-обертки для обратной совместимости

// RussianSoundex возвращает Soundex код для русского языка (обратная совместимость)
func RussianSoundex(text string) string {
	s := NewSoundex()
	return s.Encode(text)
}

// RussianMetaphone возвращает Metaphone код для русского языка (обратная совместимость)
func RussianMetaphone(text string) string {
	m := NewMetaphone()
	return m.Encode(text)
}

// PhoneticSimilarity вычисляет фонетическое сходство двух строк (обратная совместимость)
func PhoneticSimilarity(s1, s2 string, method string) float64 {
	pm := NewPhoneticMatcher()
	
	switch method {
	case "soundex":
		s := NewSoundex()
		return s.Similarity(s1, s2)
	case "metaphone":
		m := NewMetaphone()
		return m.Similarity(s1, s2)
	case "phonetic_hash", "phonetic":
		return pm.Similarity(s1, s2)
	default:
		return pm.Similarity(s1, s2)
	}
}

// ImprovedRussianPhoneticHash возвращает улучшенный фонетический хэш (обратная совместимость)
// Использует комбинацию Soundex и Metaphone
func ImprovedRussianPhoneticHash(text string) string {
	s := NewSoundex()
	m := NewMetaphone()
	soundexCode := s.Encode(text)
	metaphoneCode := m.Encode(text)
	
	// Комбинируем коды
	combined := soundexCode + metaphoneCode
	if len(combined) > 8 {
		return combined[:8]
	}
	return combined
}
