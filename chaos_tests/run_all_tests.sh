#!/bin/bash
# Скрипт для запуска всех Chaos Monkey тестов

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║     Chaos Monkey Backend Testing - Все тесты              ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Проверка доступности сервера
if ! check_server; then
    log ERROR "Сервер недоступен. Убедитесь, что сервер запущен на $BASE_URL"
    exit 1
fi

# Создаем сводный отчет
SUMMARY_REPORT="$REPORT_DIR/chaos_test_summary_$(date +%Y%m%d_%H%M%S).md"
{
    echo "# Chaos Monkey Backend Testing - Сводный отчет"
    echo ""
    echo "**Дата:** $(date)"
    echo "**Сервер:** $BASE_URL"
    echo ""
    echo "## Выполненные тесты"
    echo ""
} > "$SUMMARY_REPORT"

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Функция для запуска теста
run_test() {
    local test_name=$1
    local test_script=$2
    
    ((TESTS_RUN++))
    log INFO "=== Запуск теста: $test_name ==="
    
    if [ -f "$SCRIPT_DIR/$test_script" ]; then
        if bash "$SCRIPT_DIR/$test_script"; then
            log SUCCESS "Тест '$test_name' завершен успешно"
            ((TESTS_PASSED++))
            {
                echo "### $test_name"
                echo ""
                echo "✅ **Статус:** Пройден"
                echo ""
            } >> "$SUMMARY_REPORT"
            return 0
        else
            log ERROR "Тест '$test_name' завершился с ошибкой"
            ((TESTS_FAILED++))
            {
                echo "### $test_name"
                echo ""
                echo "❌ **Статус:** Провален"
                echo ""
            } >> "$SUMMARY_REPORT"
            return 1
        fi
    else
        log ERROR "Скрипт теста не найден: $test_script"
        ((TESTS_FAILED++))
        {
            echo "### $test_name"
            echo ""
            echo "⚠️ **Статус:** Скрипт не найден"
            echo ""
        } >> "$SUMMARY_REPORT"
        return 1
    fi
}

# Запускаем тесты
echo ""
log INFO "Начало выполнения всех тестов"
echo ""

# Тест 1: Конкурентные обновления конфигурации
run_test "Конкурентные обновления конфигурации" "test_concurrent_config.sh"
echo ""

# Тест 2: Нормализация с невалидными данными
run_test "Нормализация с невалидными данными" "test_invalid_normalization.sh"
echo ""

# Тест 3: Устойчивость к сбоям AI
run_test "Устойчивость к сбоям AI сервиса" "test_ai_failure.sh"
echo ""

# Тест 4: Работа с большими объемами
run_test "Работа с большими объемами данных" "test_large_data.sh"
echo ""

# Завершаем сводный отчет
{
    echo "## Итоги"
    echo ""
    echo "**Всего тестов:** $TESTS_RUN"
    echo "**Пройдено:** $TESTS_PASSED"
    echo "**Провалено:** $TESTS_FAILED"
    echo ""
    if [ $TESTS_FAILED -eq 0 ]; then
        echo "✅ Все тесты пройдены успешно!"
    else
        echo "⚠️ Некоторые тесты провалены. Проверьте детальные отчеты в $REPORT_DIR"
    fi
    echo ""
    echo "## Детальные отчеты"
    echo ""
    echo "Индивидуальные отчеты по каждому тесту доступны в: $REPORT_DIR"
    echo ""
    echo "## Следующие шаги"
    echo ""
    echo "1. Просмотрите детальные отчеты по каждому тесту"
    echo "2. Проверьте логи в: $LOG_DIR"
    echo "3. Исправьте найденные проблемы"
    echo "4. Повторите тесты после исправлений"
    echo ""
} >> "$SUMMARY_REPORT"

# Выводим итоги
echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                    Итоги тестирования                     ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
log INFO "Всего тестов: $TESTS_RUN"
log SUCCESS "Пройдено: $TESTS_PASSED"
if [ $TESTS_FAILED -gt 0 ]; then
    log ERROR "Провалено: $TESTS_FAILED"
else
    log SUCCESS "Провалено: $TESTS_FAILED"
fi
echo ""
log INFO "Сводный отчет: $SUMMARY_REPORT"
log INFO "Детальные отчеты: $REPORT_DIR"
log INFO "Логи: $LOG_DIR"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    log SUCCESS "Все тесты пройдены успешно!"
    exit 0
else
    log WARNING "Некоторые тесты провалены. Проверьте детальные отчеты."
    exit 1
fi

