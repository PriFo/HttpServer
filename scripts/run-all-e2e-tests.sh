#!/bin/bash
# Скрипт для запуска всех E2E тестов
# Использование: ./scripts/run-all-e2e-tests.sh [--headed] [--ui] [--debug] [--grep "pattern"]

set -e

HEADED=false
UI=false
DEBUG=false
GREP=""
PROJECT="frontend"

# Парсинг аргументов
while [[ $# -gt 0 ]]; do
    case $1 in
        --headed)
            HEADED=true
            shift
            ;;
        --ui)
            UI=true
            shift
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        --grep)
            GREP="$2"
            shift 2
            ;;
        --project)
            PROJECT="$2"
            shift 2
            ;;
        *)
            echo "Неизвестный аргумент: $1"
            exit 1
            ;;
    esac
done

echo "🚀 Запуск всех E2E тестов..."
echo ""

# Проверяем, что мы в правильной директории
if [ ! -f "package.json" ]; then
    echo "❌ Ошибка: package.json не найден. Запустите скрипт из корня проекта."
    exit 1
fi

# Проверяем, что фронтенд существует
if [ ! -d "$PROJECT" ]; then
    echo "❌ Ошибка: Директория $PROJECT не найдена."
    exit 1
fi

# Переходим в директорию фронтенда
cd "$PROJECT"

# Проверяем наличие Playwright
if ! npm list @playwright/test >/dev/null 2>&1; then
    echo "📦 Установка Playwright..."
    npm install
    npx playwright install
fi

# Формируем команду
CMD="npx playwright test tests/e2e"

if [ "$UI" = true ]; then
    CMD="$CMD --ui"
    echo "🎨 Запуск в UI режиме..."
elif [ "$DEBUG" = true ]; then
    CMD="$CMD --debug"
    echo "🐛 Запуск в режиме отладки..."
elif [ "$HEADED" = true ]; then
    CMD="$CMD --headed"
    echo "👀 Запуск в видимом режиме..."
fi

if [ -n "$GREP" ]; then
    CMD="$CMD --grep \"$GREP\""
    echo "🔍 Фильтр: $GREP"
fi

echo ""
echo "Выполняется: $CMD"
echo ""

# Запускаем тесты
eval $CMD

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ Все тесты прошли успешно!"
else
    echo ""
    echo "❌ Некоторые тесты провалились. Код выхода: $EXIT_CODE"
    echo "📊 Просмотр отчета: npx playwright show-report"
fi

exit $EXIT_CODE

