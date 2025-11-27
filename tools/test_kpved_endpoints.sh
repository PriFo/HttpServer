#!/bin/bash
# Скрипт для тестирования endpoints КПВЭД после перезапуска сервера

BASE_URL="http://localhost:9999"

echo "=== Тестирование endpoints КПВЭД ==="
echo ""

# Тест 1: Статистика КПВЭД
echo "1. Проверка GET /api/kpved/stats"
STATS_RESPONSE=$(curl -s "$BASE_URL/api/kpved/stats")
if [ $? -eq 0 ]; then
    echo "✓ Endpoint доступен"
    echo "Ответ:"
    echo "$STATS_RESPONSE" | python -m json.tool 2>/dev/null || echo "$STATS_RESPONSE"
else
    echo "✗ Endpoint недоступен (возможно, сервер не запущен или endpoint не зарегистрирован)"
fi
echo ""

# Тест 2: Проверка структуры ответа
echo "2. Проверка структуры ответа статистики"
TOTAL=$(echo "$STATS_RESPONSE" | grep -o '"total":[0-9]*' | grep -o '[0-9]*')
if [ -n "$TOTAL" ]; then
    echo "✓ Всего кодов КПВЭД: $TOTAL"
    if [ "$TOTAL" -gt 0 ]; then
        echo "✓ Данные присутствуют в базе"
    else
        echo "⚠ Данные отсутствуют в базе"
    fi
else
    echo "✗ Не удалось получить количество кодов"
fi
echo ""

echo "=== Тестирование завершено ==="

