#!/bin/bash
# Скрипт для запуска E2E тестов
# Использование: ./scripts/run-e2e-tests.sh [опции]

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Параметры по умолчанию
TEST_FILE=""
BROWSER="chromium"
HEADED=false
DEBUG=false
UI=false

# Парсинг аргументов
while [[ $# -gt 0 ]]; do
  case $1 in
    -f|--file)
      TEST_FILE="$2"
      shift 2
      ;;
    -b|--browser)
      BROWSER="$2"
      shift 2
      ;;
    --headed)
      HEADED=true
      shift
      ;;
    --debug)
      DEBUG=true
      shift
      ;;
    --ui)
      UI=true
      shift
      ;;
    -h|--help)
      echo "Использование: $0 [опции]"
      echo ""
      echo "Опции:"
      echo "  -f, --file <путь>    Запустить конкретный тест"
      echo "  -b, --browser <name> Браузер (chromium, firefox, webkit)"
      echo "  --headed             Запустить в видимом режиме"
      echo "  --debug              Запустить в режиме отладки"
      echo "  --ui                 Запустить с UI"
      echo "  -h, --help           Показать эту справку"
      exit 0
      ;;
    *)
      echo "Неизвестная опция: $1"
      exit 1
      ;;
  esac
done

echo -e "${GREEN}🚀 Запуск E2E тестов${NC}"
echo ""

# Проверка зависимостей
echo -e "${YELLOW}Проверка зависимостей...${NC}"

# Проверяем, что Playwright установлен
if ! command -v npx &> /dev/null; then
  echo -e "${RED}❌ npx не найден. Установите Node.js${NC}"
  exit 1
fi

# Проверяем, что бэкенд запущен
echo -e "${YELLOW}Проверка бэкенда...${NC}"
if ! curl -s http://127.0.0.1:9999/health > /dev/null 2>&1; then
  echo -e "${RED}❌ Бэкенд не доступен на http://127.0.0.1:9999${NC}"
  echo -e "${YELLOW}💡 Запустите бэкенд: docker-compose up -d backend${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Бэкенд доступен${NC}"

# Проверяем, что фронтенд запущен
echo -e "${YELLOW}Проверка фронтенда...${NC}"
if ! curl -s http://localhost:3000 > /dev/null 2>&1; then
  echo -e "${RED}❌ Фронтенд не доступен на http://localhost:3000${NC}"
  echo -e "${YELLOW}💡 Запустите фронтенд: cd frontend && npm run dev${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Фронтенд доступен${NC}"

echo ""

# Формируем команду
CMD="npx playwright test"

if [ -n "$TEST_FILE" ]; then
  CMD="$CMD $TEST_FILE"
fi

CMD="$CMD --project=$BROWSER"

if [ "$HEADED" = true ]; then
  CMD="$CMD --headed"
fi

if [ "$DEBUG" = true ]; then
  CMD="$CMD --debug"
fi

if [ "$UI" = true ]; then
  CMD="$CMD --ui"
fi

echo -e "${GREEN}Выполняем: $CMD${NC}"
echo ""

# Запускаем тесты
cd frontend
eval $CMD

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
  echo ""
  echo -e "${GREEN}✅ Все тесты прошли успешно!${NC}"
else
  echo ""
  echo -e "${RED}❌ Некоторые тесты провалились${NC}"
fi

exit $EXIT_CODE

