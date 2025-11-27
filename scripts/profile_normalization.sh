#!/bin/bash
# Скрипт для профилирования нормализации контрагентов с помощью pprof

set -e

PORT=${PORT:-9999}
DURATION=${DURATION:-30}
OUTPUT_DIR=${OUTPUT_DIR:-./profiles}

echo "=== Профилирование нормализации контрагентов ==="
echo "Порт: $PORT"
echo "Длительность: ${DURATION} секунд"
echo "Выходная директория: $OUTPUT_DIR"
echo ""

# Создаем директорию для профилей
mkdir -p "$OUTPUT_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo "1. Генерация CPU профиля..."
go tool pprof -proto -output="$OUTPUT_DIR/cpu_profile_${TIMESTAMP}.pb.gz" \
    "http://localhost:${PORT}/debug/pprof/profile?seconds=${DURATION}" || {
    echo "ОШИБКА: Не удалось получить CPU профиль. Убедитесь, что:"
    echo "  - Сервер запущен на порту $PORT"
    echo "  - Эндпоинт /debug/pprof/ доступен"
    echo "  - Нормализация выполняется во время профилирования"
    exit 1
}

echo "2. Генерация Memory профиля..."
go tool pprof -proto -output="$OUTPUT_DIR/mem_profile_${TIMESTAMP}.pb.gz" \
    "http://localhost:${PORT}/debug/pprof/heap" || {
    echo "ОШИБКА: Не удалось получить Memory профиль"
    exit 1
}

echo "3. Генерация Goroutine профиля..."
go tool pprof -proto -output="$OUTPUT_DIR/goroutine_profile_${TIMESTAMP}.pb.gz" \
    "http://localhost:${PORT}/debug/pprof/goroutine" || {
    echo "ОШИБКА: Не удалось получить Goroutine профиль"
    exit 1
}

echo ""
echo "=== Профилирование завершено ==="
echo "CPU профиль: $OUTPUT_DIR/cpu_profile_${TIMESTAMP}.pb.gz"
echo "Memory профиль: $OUTPUT_DIR/mem_profile_${TIMESTAMP}.pb.gz"
echo "Goroutine профиль: $OUTPUT_DIR/goroutine_profile_${TIMESTAMP}.pb.gz"
echo ""
echo "Для анализа используйте:"
echo "  go tool pprof $OUTPUT_DIR/cpu_profile_${TIMESTAMP}.pb.gz"
echo "  # В интерактивной оболочке:"
echo "  # (pprof) top10"
echo "  # (pprof) web"
echo "  # (pprof) list FunctionName"

