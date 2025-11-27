#!/bin/bash
# Общие утилиты для Chaos Monkey тестов

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Настройки по умолчанию
BASE_URL="${BASE_URL:-http://localhost:9999}"
LOG_DIR="${LOG_DIR:-./chaos_tests/logs}"
REPORT_DIR="${REPORT_DIR:-./chaos_tests/reports}"

# Проверка зависимостей
check_dependencies() {
    local missing=0
    
    if ! command -v curl &> /dev/null; then
        echo "Ошибка: curl не установлен" >&2
        missing=1
    fi
    
    if ! command -v jq &> /dev/null; then
        echo "Ошибка: jq не установлен" >&2
        missing=1
    fi
    
    if ! command -v bc &> /dev/null; then
        echo "Ошибка: bc не установлен" >&2
        missing=1
    fi
    
    if [ $missing -eq 1 ]; then
        echo "Установите недостающие зависимости и повторите попытку." >&2
        return 1
    fi
    
    return 0
}

# Проверяем зависимости при загрузке
if ! check_dependencies; then
    exit 1
fi

# Создаем директории для логов и отчетов
mkdir -p "$LOG_DIR" "$REPORT_DIR"

# Функция логирования
log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local log_file="$LOG_DIR/chaos_test_$(date +%Y%m%d).log"
    
    case $level in
        INFO)
            echo -e "${CYAN}[INFO]${NC} $message" | tee -a "$log_file"
            ;;
        SUCCESS)
            echo -e "${GREEN}[SUCCESS]${NC} $message" | tee -a "$log_file"
            ;;
        WARNING)
            echo -e "${YELLOW}[WARNING]${NC} $message" | tee -a "$log_file"
            ;;
        ERROR)
            echo -e "${RED}[ERROR]${NC} $message" | tee -a "$log_file"
            ;;
        *)
            echo "[$level] $message" | tee -a "$log_file"
            ;;
    esac
}

# Функция проверки доступности сервера
check_server() {
    log INFO "Проверка доступности сервера на $BASE_URL"
    if curl -s -f --max-time 5 "$BASE_URL/api/config" > /dev/null 2>&1; then
        log SUCCESS "Сервер доступен"
        return 0
    else
        log ERROR "Сервер недоступен на $BASE_URL"
        return 1
    fi
}

# Функция выполнения HTTP запроса с логированием
http_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local headers=$4
    local output_file="$LOG_DIR/response_$(date +%s%N).json"
    
    local curl_cmd="curl -s -w '\n%{http_code}\n%{time_total}' -X $method"
    
    if [ -n "$headers" ]; then
        curl_cmd="$curl_cmd -H \"$headers\""
    fi
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/json' -d '$data'"
    fi
    
    curl_cmd="$curl_cmd -o '$output_file' '$BASE_URL$endpoint'"
    
    local response=$(eval $curl_cmd 2>&1)
    local http_code=$(echo "$response" | tail -n 2 | head -n 1)
    local time_total=$(echo "$response" | tail -n 1)
    local body=$(cat "$output_file" 2>/dev/null || echo "")
    
    echo "$http_code|$time_total|$output_file|$body"
}

# Функция проверки ответа
check_response() {
    local response_data=$1
    local expected_status=${2:-200}
    
    local http_code=$(echo "$response_data" | cut -d'|' -f1)
    local time_total=$(echo "$response_data" | cut -d'|' -f2)
    local body=$(echo "$response_data" | cut -d'|' -f4-)
    
    if [ "$http_code" -eq "$expected_status" ]; then
        log SUCCESS "HTTP $http_code (время: ${time_total}s)"
        return 0
    else
        log ERROR "Ожидался статус $expected_status, получен $http_code"
        log ERROR "Тело ответа: $body"
        return 1
    fi
}

# Функция проверки наличия ошибки в логах
check_logs_for_error() {
    local error_pattern=$1
    local log_file=${2:-"$LOG_DIR/chaos_test_$(date +%Y%m%d).log"}
    
    if grep -i "$error_pattern" "$log_file" > /dev/null 2>&1; then
        log WARNING "Найдена ошибка в логах: $error_pattern"
        grep -i "$error_pattern" "$log_file" | tail -5
        return 0
    else
        log INFO "Ошибка '$error_pattern' не найдена в логах"
        return 1
    fi
}

# Функция мониторинга ресурсов процесса
monitor_resources() {
    local process_name=$1
    local duration=${2:-60}
    local interval=${3:-5}
    local output_file="$REPORT_DIR/resources_$(date +%Y%m%d_%H%M%S).csv"
    
    log INFO "Мониторинг ресурсов процесса '$process_name' в течение ${duration}с (интервал ${interval}с)"
    
    echo "timestamp,cpu_percent,memory_mb,rss_mb,vms_mb" > "$output_file"
    
    local end_time=$(($(date +%s) + duration))
    while [ $(date +%s) -lt $end_time ]; do
        local pid=$(pgrep -f "$process_name" | head -1)
        if [ -n "$pid" ]; then
            local stats=$(ps -p "$pid" -o %cpu,rss,vsz --no-headers 2>/dev/null)
            if [ -n "$stats" ]; then
                local cpu=$(echo "$stats" | awk '{print $1}')
                local rss_kb=$(echo "$stats" | awk '{print $2}')
                local vms_kb=$(echo "$stats" | awk '{print $3}')
                local rss_mb=$(echo "scale=2; $rss_kb / 1024" | bc)
                local vms_mb=$(echo "scale=2; $vms_kb / 1024" | bc)
                local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
                echo "$timestamp,$cpu,$rss_mb,$rss_mb,$vms_mb" >> "$output_file"
                log INFO "CPU: ${cpu}%, Memory: ${rss_mb}MB (RSS), ${vms_mb}MB (VMS)"
            fi
        else
            log WARNING "Процесс '$process_name' не найден"
        fi
        sleep "$interval"
    done
    
    log SUCCESS "Мониторинг завершен. Данные сохранены в $output_file"
    echo "$output_file"
}

# Функция получения текущей конфигурации
get_config() {
    local response=$(http_request "GET" "/api/config" "" "")
    local http_code=$(echo "$response" | cut -d'|' -f1)
    local body=$(echo "$response" | cut -d'|' -f4-)
    
    if [ "$http_code" -eq 200 ]; then
        echo "$body"
        return 0
    else
        log ERROR "Не удалось получить конфигурацию: HTTP $http_code"
        return 1
    fi
}

# Функция сохранения конфигурации
save_config() {
    local config_json=$1
    local response=$(http_request "PUT" "/api/config" "$config_json" "")
    local http_code=$(echo "$response" | cut -d'|' -f1)
    
    if [ "$http_code" -eq 200 ]; then
        log SUCCESS "Конфигурация сохранена"
        return 0
    else
        log ERROR "Не удалось сохранить конфигурацию: HTTP $http_code"
        return 1
    fi
}

# Функция ожидания завершения операции
wait_for_completion() {
    local check_func=$1
    local timeout=${2:-300}
    local interval=${3:-5}
    local start_time=$(date +%s)
    
    log INFO "Ожидание завершения операции (таймаут: ${timeout}с)"
    
    while [ $(($(date +%s) - start_time)) -lt $timeout ]; do
        if $check_func; then
            log SUCCESS "Операция завершена"
            return 0
        fi
        sleep "$interval"
    done
    
    log ERROR "Таймаут ожидания операции"
    return 1
}

# Функция запуска параллельных запросов
run_parallel_requests() {
    local count=$1
    local request_func=$2
    local output_dir="$LOG_DIR/parallel_$(date +%s)"
    mkdir -p "$output_dir"
    
    log INFO "Запуск $count параллельных запросов"
    
    local pids=()
    for i in $(seq 1 $count); do
        (
            local result=$($request_func $i)
            echo "$result" > "$output_dir/request_$i.txt"
        ) &
        pids+=($!)
    done
    
    # Ждем завершения всех процессов
    local failed=0
    for pid in "${pids[@]}"; do
        wait "$pid"
        if [ $? -ne 0 ]; then
            ((failed++))
        fi
    done
    
    log INFO "Завершено: успешно $((count - failed)), неудачно $failed"
    echo "$output_dir"
}

# Функция анализа результатов параллельных запросов
analyze_parallel_results() {
    local results_dir=$1
    local success=0
    local failed=0
    local db_locked=0
    local http_codes=""
    
    log INFO "Анализ результатов из $results_dir"
    
    for result_file in "$results_dir"/request_*.txt; do
        if [ -f "$result_file" ]; then
            local content=$(cat "$result_file")
            local http_code=$(echo "$content" | cut -d'|' -f1)
            
            if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
                ((success++))
            else
                ((failed++))
            fi
            
            if echo "$content" | grep -i "database.*locked\|locked" > /dev/null; then
                ((db_locked++))
            fi
            
            http_codes="$http_codes $http_code"
        fi
    done
    
    log INFO "Результаты анализа:"
    log INFO "  Успешных: $success"
    log INFO "  Неудачных: $failed"
    log INFO "  Ошибок блокировки БД: $db_locked"
    log INFO "  HTTP коды: $http_codes"
    
    echo "$success|$failed|$db_locked"
}

# Функция генерации отчета
generate_report_section() {
    local title=$1
    local content=$2
    local report_file=$3
    
    {
        echo ""
        echo "## $title"
        echo ""
        echo "$content"
        echo ""
        echo "---"
    } >> "$report_file"
}

