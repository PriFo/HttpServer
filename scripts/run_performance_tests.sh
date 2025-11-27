#!/bin/bash
# Скрипт для запуска полного набора тестов производительности

set -e

echo "=== Полный набор тестов производительности нормализации ==="
echo ""

OUTPUT_DIR="./benchmark_results"
mkdir -p "$OUTPUT_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo "1. Запуск бенчмарков для разных объемов данных..."
echo ""

# Бенчмарки для разных объемов
for size in 1K 10K 50K 100K; do
    echo "  Тестирование $size записей..."
    go test -bench="BenchmarkProcessNormalization_${size}" -benchmem -count=3 \
        ./normalization > "${OUTPUT_DIR}/benchmark_${size}_${TIMESTAMP}.txt" 2>&1 || true
done

echo ""
echo "2. Запуск бенчмарков для подпроцессов..."
echo ""

# Бенчмарки для подпроцессов
go test -bench="BenchmarkDuplicateAnalysis|BenchmarkDataExtraction|BenchmarkBenchmarkLookup|BenchmarkNormalizeCounterparty" \
    -benchmem -count=3 ./normalization > "${OUTPUT_DIR}/benchmark_subprocesses_${TIMESTAMP}.txt" 2>&1 || true

echo ""
echo "3. Запуск параллельных бенчмарков..."
echo ""

go test -bench="BenchmarkProcessNormalization_Parallel" -benchmem -count=3 \
    ./normalization > "${OUTPUT_DIR}/benchmark_parallel_${TIMESTAMP}.txt" 2>&1 || true

echo ""
echo "=== Тесты завершены ==="
echo "Результаты сохранены в: $OUTPUT_DIR"
echo ""
echo "Для анализа используйте:"
echo "  cat ${OUTPUT_DIR}/benchmark_10K_${TIMESTAMP}.txt"
echo "  benchstat ${OUTPUT_DIR}/benchmark_10K_${TIMESTAMP}.txt"

