#!/bin/bash
# Скрипт для сравнения результатов бенчмарков до и после оптимизаций

set -e

BENCHMARK_PATTERN=${1:-"BenchmarkProcessNormalization"}
OUTPUT_FILE=${2:-"benchmark_comparison.txt"}

echo "=== Сравнение результатов бенчмарков ==="
echo "Паттерн: $BENCHMARK_PATTERN"
echo "Выходной файл: $OUTPUT_FILE"
echo ""

# Запускаем бенчмарки и сохраняем результаты
echo "Запуск бенчмарков..."
go test -bench="$BENCHMARK_PATTERN" -benchmem -count=5 ./normalization > "$OUTPUT_FILE" 2>&1

echo ""
echo "=== Результаты сохранены в $OUTPUT_FILE ==="
echo ""
echo "Для анализа используйте:"
echo "  benchstat $OUTPUT_FILE"
echo ""
echo "Или сравните два файла:"
echo "  benchstat before.txt after.txt"

