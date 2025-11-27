#!/bin/bash

# Автоматизированная система управления TODO
# Использование: ./automated-todo-system.sh [scan|report|stats|auto]

PROJECT_DIR="${PROJECT_DIR:-.}"
SCAN_INTERVAL="${SCAN_INTERVAL:-3600}"  # 1 час по умолчанию
TODO_DB="$PROJECT_DIR/.todos/tasks.json"
TEAM_CONFIG="$PROJECT_DIR/.todos/team.json"
REPORT_DIR="$PROJECT_DIR/.todos"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Функция сканирования
scan_todos() {
    log "🔄 Начало автоматического сканирования TODO..."
    
    # Проверяем наличие Go утилиты
    if ! command -v go &> /dev/null; then
        error "Go не установлен. Установите Go для работы системы."
        return 1
    fi
    
    # Компилируем утилиту если нужно
    if [ ! -f "$PROJECT_DIR/cmd/scan_todos/scan_todos" ]; then
        log "📦 Компиляция утилиты сканирования..."
        cd "$PROJECT_DIR" || exit 1
        go build -o cmd/scan_todos/scan_todos ./cmd/scan_todos
        if [ $? -ne 0 ]; then
            error "Ошибка компиляции утилиты"
            return 1
        fi
    fi
    
    # Запускаем сканирование
    "$PROJECT_DIR/cmd/scan_todos/scan_todos" scan "$PROJECT_DIR"
    
    if [ $? -eq 0 ]; then
        log "✅ Сканирование завершено успешно"
        
        # Автоматически генерируем отчет
        generate_report
    else
        error "Ошибка при сканировании"
        return 1
    fi
}

# Генерация отчета
generate_report() {
    log "📊 Генерация отчета..."
    
    if [ ! -f "$PROJECT_DIR/cmd/scan_todos/scan_todos" ]; then
        error "Утилита сканирования не найдена. Запустите 'scan' сначала."
        return 1
    fi
    
    "$PROJECT_DIR/cmd/scan_todos/scan_todos" report
    
    if [ $? -eq 0 ]; then
        log "✅ Отчет сгенерирован: $REPORT_DIR/dashboard.html"
    else
        error "Ошибка генерации отчета"
        return 1
    fi
}

# Показать статистику
show_stats() {
    if [ ! -f "$PROJECT_DIR/cmd/scan_todos/scan_todos" ]; then
        error "Утилита сканирования не найдена"
        return 1
    fi
    
    "$PROJECT_DIR/cmd/scan_todos/scan_todos" stats
}

# Автоматический режим (сканирование каждые N секунд)
auto_mode() {
    log "🚀 Запуск автоматического режима (интервал: ${SCAN_INTERVAL}с)"
    log "Для остановки нажмите Ctrl+C"
    
    while true; do
        scan_todos
        log "⏳ Ожидание следующего сканирования..."
        sleep "$SCAN_INTERVAL"
    done
}

# Инициализация
init_system() {
    log "🔧 Инициализация системы TODO..."
    
    # Создаем директории
    mkdir -p "$REPORT_DIR/scripts"
    
    # Создаем БД если нужно
    if [ ! -f "$TODO_DB" ]; then
        echo '{"tasks": [], "lastScan": null, "version": "1.0.0"}' > "$TODO_DB"
        log "✅ Создана база данных задач"
    fi
    
    # Создаем конфиг команды если нужно
    if [ ! -f "$TEAM_CONFIG" ]; then
        log "⚠️  Конфиг команды не найден. Создайте $TEAM_CONFIG"
    fi
}

# Основная функция
main() {
    local command="${1:-help}"
    
    case "$command" in
        scan)
            init_system
            scan_todos
            ;;
        report)
            init_system
            generate_report
            ;;
        stats)
            show_stats
            ;;
        auto)
            init_system
            auto_mode
            ;;
        help|--help|-h)
            echo "Автоматизированная система управления TODO"
            echo ""
            echo "Использование: $0 [команда]"
            echo ""
            echo "Команды:"
            echo "  scan    - Сканировать проект на TODO"
            echo "  report  - Сгенерировать отчет"
            echo "  stats   - Показать статистику"
            echo "  auto    - Автоматический режим (сканирование каждые $SCAN_INTERVAL секунд)"
            echo "  help    - Показать эту справку"
            echo ""
            echo "Переменные окружения:"
            echo "  PROJECT_DIR  - Директория проекта (по умолчанию: .)"
            echo "  SCAN_INTERVAL - Интервал сканирования в секундах (по умолчанию: 3600)"
            ;;
        *)
            error "Неизвестная команда: $command"
            echo "Используйте '$0 help' для справки"
            exit 1
            ;;
    esac
}

main "$@"

