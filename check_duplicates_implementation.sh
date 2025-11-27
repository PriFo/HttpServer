#!/bin/bash

# Скрипт для автоматической проверки реализации методов обнаружения дублей
# Дата создания: 2025-01-20

echo "=========================================="
echo "Проверка реализации методов обнаружения дублей"
echo "=========================================="
echo ""

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Счетчики
TOTAL=0
PASSED=0
FAILED=0
WARNING=0

# Функция проверки
check_implementation() {
    local name=$1
    local pattern=$2
    local file=$3
    
    TOTAL=$((TOTAL + 1))
    
    if grep -r "$pattern" "$file" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $name"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}✗${NC} $name"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# Функция проверки с предупреждением
check_with_warning() {
    local name=$1
    local pattern=$2
    local file=$3
    
    TOTAL=$((TOTAL + 1))
    
    if grep -r "$pattern" "$file" > /dev/null 2>&1; then
        echo -e "${YELLOW}⚠${NC} $name (частично)"
        WARNING=$((WARNING + 1))
        return 0
    else
        echo -e "${RED}✗${NC} $name"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

echo "ГРУППА 1: Предобработка текста"
echo "-------------------------------"
check_implementation "Приведение к регистру" "strings.ToLower" "normalization/"
check_implementation "Удаление пробелов" "TrimSpace\|strings.Fields" "normalization/"
check_implementation "Удаление пунктуации" "regexp.*ReplaceAllString" "normalization/"
check_with_warning "Нормализация Unicode" "rune\|unicode" "normalization/"
echo ""

echo "ГРУППА 2: Лингвистические алгоритмы"
echo "-----------------------------------"
check_implementation "Стемминг" "Stem\|snowball" "normalization/algorithms/"
check_with_warning "Лемматизация" "Lemmatize\|pymorphy" "normalization/"
check_implementation "Удаление стоп-слов" "stopWords\|stop-words" "normalization/"
echo ""

echo "ГРУППА 3: Алгоритмы сравнения строк"
echo "------------------------------------"
check_implementation "Расстояние Левенштейна" "levenshteinDistance\|Levenshtein" "normalization/"
check_implementation "Расстояние Дамерау-Левенштейна" "DamerauLevenshtein" "normalization/"
check_implementation "N-граммы" "Bigram\|Trigram\|NGram" "normalization/"
check_implementation "Индекс Жаккара" "Jaccard" "normalization/"
check_implementation "Soundex" "Soundex" "normalization/algorithms/"
check_implementation "Metaphone" "Metaphone" "normalization/algorithms/"
echo ""

echo "ГРУППА 4: Структурирование данных"
echo "----------------------------------"
check_implementation "Регулярные выражения" "regexp\." "normalization/"
check_implementation "Токенизация" "tokenize\|Tokenize" "normalization/"
check_with_warning "NER" "NER\|NamedEntity\|EntityRecognition" "normalization/"
echo ""

echo "ГРУППА 5: Машинное обучение"
echo "---------------------------"
check_implementation "LLM через API" "AINormalizer\|LLM" "normalization/"
# Seq2Seq, BERT, BiLSTM, Random Forest, Gradient Boosting - не реализованы
echo -e "${RED}✗${NC} Seq2Seq модели"
FAILED=$((FAILED + 1))
TOTAL=$((TOTAL + 1))
echo -e "${RED}✗${NC} BiLSTM"
FAILED=$((FAILED + 1))
TOTAL=$((TOTAL + 1))
echo -e "${RED}✗${NC} BERT/Трансформеры"
FAILED=$((FAILED + 1))
TOTAL=$((TOTAL + 1))
echo -e "${RED}✗${NC} Random Forest"
FAILED=$((FAILED + 1))
TOTAL=$((TOTAL + 1))
echo -e "${RED}✗${NC} Gradient Boosting"
FAILED=$((FAILED + 1))
TOTAL=$((TOTAL + 1))
echo ""

echo "ГРУППА 6: Консолидация данных"
echo "------------------------------"
check_implementation "Группировка дубликатов" "mergeOverlappingGroups\|groupDuplicates" "normalization/"
check_implementation "Выбор мастер-записи" "selectMasterRecord\|master.*record" "normalization/"
echo ""

echo "ГРУППА 7: Метрики оценки"
echo "------------------------"
check_implementation "Precision" "Precision" "normalization/"
check_implementation "Recall" "Recall" "normalization/"
check_implementation "F1-мера" "F1Score\|F1.*Score" "normalization/"
check_implementation "Ошибки первого рода" "FalsePositive\|FPR" "normalization/"
check_implementation "Ошибки второго рода" "FalseNegative\|FNR" "normalization/"
check_implementation "Индекс Жаккара" "Jaccard" "normalization/"
check_implementation "ROC-кривая и AUC" "ROC\|AUC\|CalculateROC" "normalization/"
echo ""

echo "=========================================="
echo "ИТОГИ ПРОВЕРКИ"
echo "=========================================="
echo "Всего проверок: $TOTAL"
echo -e "${GREEN}Успешно: $PASSED${NC}"
echo -e "${YELLOW}С предупреждениями: $WARNING${NC}"
echo -e "${RED}Не реализовано: $FAILED${NC}"
echo ""

# Вычисляем процент
if [ $TOTAL -gt 0 ]; then
    PERCENT=$((PASSED * 100 / TOTAL))
    echo "Покрытие: $PERCENT%"
    
    if [ $PERCENT -ge 85 ]; then
        echo -e "${GREEN}✓ Отличное покрытие!${NC}"
    elif [ $PERCENT -ge 70 ]; then
        echo -e "${YELLOW}⚠ Хорошее покрытие, есть что улучшить${NC}"
    else
        echo -e "${RED}✗ Низкое покрытие, требуется доработка${NC}"
    fi
fi

echo ""
echo "=========================================="
echo "РЕКОМЕНДАЦИИ"
echo "=========================================="

if [ $FAILED -gt 0 ]; then
    echo "Критичные улучшения:"
    echo "  1. Добавить лемматизацию (заменить стемминг)"
    echo "  2. Улучшить NER (полноценное распознавание)"
    echo "  3. Добавить префиксную фильтрацию"
    echo ""
fi

if [ $WARNING -gt 0 ]; then
    echo "Улучшения с предупреждениями:"
    echo "  1. Полная нормализация Unicode"
    echo "  2. Полноценный NER"
    echo ""
fi

echo "Для детальной информации см. документы:"
echo "  - DUPLICATES_CHECK_SUMMARY.md"
echo "  - DUPLICATES_IMPROVEMENT_PLAN.md"
echo ""

