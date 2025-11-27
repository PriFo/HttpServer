package algorithms

import (
	"unicode"
)

// DamerauLevenshtein вычисляет расстояние Дамерау-Левенштейна
// Это улучшенная версия расстояния Левенштейна, которая также учитывает
// транспозицию (перестановку) двух соседних символов
type DamerauLevenshtein struct{}

// NewDamerauLevenshtein создает новый вычислитель расстояния Дамерау-Левенштейна
func NewDamerauLevenshtein() *DamerauLevenshtein {
	return &DamerauLevenshtein{}
}

// Distance вычисляет расстояние Дамерау-Левенштейна между двумя строками
// Возвращает минимальное количество операций (вставка, удаление, замена, транспозиция)
// для преобразования одной строки в другую
func (dl *DamerauLevenshtein) Distance(str1, str2 string) int {
	r1 := []rune(str1)
	r2 := []rune(str2)
	len1 := len(r1)
	len2 := len(r2)

	// Крайние случаи
	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Создаем матрицу для динамического программирования
	// Размер (len1+2) x (len2+2) для учета границ
	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}

	// Используем оптимизированный алгоритм с ограниченным размером
	// Создаем матрицу размером (len1+2) x (len2+2)
	matrix := make([][]int, len1+2)
	for i := range matrix {
		matrix[i] = make([]int, len2+2)
	}

	// Инициализация: максимальное расстояние
	maxDist := len1 + len2
	matrix[0][0] = maxDist

	// Инициализация первой строки и столбца
	for i := 0; i <= len1; i++ {
		matrix[i+1][0] = maxDist
		matrix[i+1][1] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j+1] = maxDist
		matrix[1][j+1] = j
	}

	// Словарь для отслеживания последнего вхождения каждого символа
	da := make(map[rune]int)

	// Заполняем матрицу
	for i := 1; i <= len1; i++ {
		db := 0
		for j := 1; j <= len2; j++ {
			i1 := da[r2[j-1]]
			j1 := db
			cost := 1

			if r1[i-1] == r2[j-1] {
				cost = 0
				db = j
			}

			// Вычисляем минимальную стоимость операций
			matrix[i+1][j+1] = min4(
				matrix[i+1][j]+1,     // вставка
				matrix[i][j+1]+1,     // удаление
				matrix[i][j]+cost,    // замена
				matrix[i1][j1]+(i-i1-1)+1+(j-j1-1), // транспозиция
			)
		}
		da[r1[i-1]] = i
	}

	return matrix[len1+1][len2+1]
}

// Similarity вычисляет схожесть двух строк на основе расстояния Дамерау-Левенштейна
// Возвращает значение от 0.0 (полностью разные) до 1.0 (идентичные)
func (dl *DamerauLevenshtein) Similarity(str1, str2 string) float64 {
	if str1 == "" && str2 == "" {
		return 1.0
	}

	distance := dl.Distance(str1, str2)
	maxLen := len([]rune(str1))
	if len([]rune(str2)) > maxLen {
		maxLen = len([]rune(str2))
	}

	if maxLen == 0 {
		return 1.0
	}

	// Схожесть = 1 - (расстояние / максимальная длина)
	similarity := 1.0 - float64(distance)/float64(maxLen)
	if similarity < 0.0 {
		similarity = 0.0
	}

	return similarity
}

// NormalizedDistance возвращает нормализованное расстояние (0.0 - 1.0)
func (dl *DamerauLevenshtein) NormalizedDistance(str1, str2 string) float64 {
	if str1 == "" && str2 == "" {
		return 0.0
	}

	distance := dl.Distance(str1, str2)
	maxLen := len([]rune(str1))
	if len([]rune(str2)) > maxLen {
		maxLen = len([]rune(str2))
	}

	if maxLen == 0 {
		return 0.0
	}

	return float64(distance) / float64(maxLen)
}

// min4 возвращает минимум из четырех чисел
func min4(a, b, c, d int) int {
	min := a
	if b < min {
		min = b
	}
	if c < min {
		min = c
	}
	if d < min {
		min = d
	}
	return min
}

// IsSimilar проверяет, являются ли две строки похожими
// Использует порог для определения схожести
func (dl *DamerauLevenshtein) IsSimilar(str1, str2 string, threshold float64) bool {
	similarity := dl.Similarity(str1, str2)
	return similarity >= threshold
}

// DistanceRunes вычисляет расстояние для рун (более эффективно для Unicode)
func (dl *DamerauLevenshtein) DistanceRunes(r1, r2 []rune) int {
	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}

	matrix := make([][]int, len1+2)
	for i := range matrix {
		matrix[i] = make([]int, len2+2)
	}

	maxDist := len1 + len2
	matrix[0][0] = maxDist

	for i := 0; i <= len1; i++ {
		matrix[i+1][0] = maxDist
		matrix[i+1][1] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j+1] = maxDist
		matrix[1][j+1] = j
	}

	da := make(map[rune]int)

	for i := 1; i <= len1; i++ {
		db := 0
		for j := 1; j <= len2; j++ {
			i1 := da[r2[j-1]]
			j1 := db
			cost := 1

			if r1[i-1] == r2[j-1] {
				cost = 0
				db = j
			}

			matrix[i+1][j+1] = min4(
				matrix[i+1][j]+1,
				matrix[i][j+1]+1,
				matrix[i][j]+cost,
				matrix[i1][j1]+(i-i1-1)+1+(j-j1-1),
			)
		}
		da[r1[i-1]] = i
	}

	return matrix[len1+1][len2+1]
}

// NormalizeString нормализует строку для сравнения
// Приводит к нижнему регистру и удаляет пробелы
func (dl *DamerauLevenshtein) NormalizeString(str string) string {
	var result []rune
	for _, r := range str {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, unicode.ToLower(r))
		}
	}
	return string(result)
}

// SimilarityNormalized вычисляет схожесть с предварительной нормализацией строк
func (dl *DamerauLevenshtein) SimilarityNormalized(str1, str2 string) float64 {
	norm1 := dl.NormalizeString(str1)
	norm2 := dl.NormalizeString(str2)
	return dl.Similarity(norm1, norm2)
}

