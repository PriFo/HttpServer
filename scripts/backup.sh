#!/bin/bash

# Скрипт для автоматического резервного копирования баз данных
# Можно запускать через cron для регулярных бэкапов

set -e

# Конфигурация
BASE_URL="${BASE_URL:-http://localhost:9999}"
BACKUP_DIR="${BACKUP_DIR:-./data/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
LOG_FILE="${LOG_FILE:-./logs/backup.log}"

# Создаем директории
mkdir -p "$BACKUP_DIR"
mkdir -p "$(dirname "$LOG_FILE")"

# Функция для логирования
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Функция для очистки старых бэкапов
cleanup_old_backups() {
    log "Очистка старых резервных копий (старше $RETENTION_DAYS дней)..."
    find "$BACKUP_DIR" -type f -name "backup_*.zip" -mtime +$RETENTION_DAYS -delete
    find "$BACKUP_DIR" -type d -name "backup_*" -mtime +$RETENTION_DAYS -exec rm -rf {} + 2>/dev/null || true
    log "Очистка завершена"
}

# Функция для создания резервной копии через API
create_backup() {
    local include_main="${1:-true}"
    local include_uploads="${2:-true}"
    local include_service="${3:-false}"
    local format="${4:-both}"
    
    log "Создание резервной копии..."
    log "  Include main: $include_main"
    log "  Include uploads: $include_uploads"
    log "  Include service: $include_service"
    log "  Format: $format"
    
    # Формируем JSON запрос
    local json_data=$(cat <<EOF
{
  "include_main": $include_main,
  "include_uploads": $include_uploads,
  "include_service": $include_service,
  "format": "$format"
}
EOF
)
    
    # Отправляем запрос
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$json_data" \
        "$BASE_URL/api/databases/backup" 2>&1)
    
    if [ $? -ne 0 ]; then
        log "ОШИБКА: Не удалось создать резервную копию"
        log "Response: $response"
        return 1
    fi
    
    # Проверяем ответ
    if echo "$response" | grep -q '"success":true'; then
        log "Резервная копия успешно создана"
        echo "$response" | tee -a "$LOG_FILE"
        return 0
    else
        log "ОШИБКА: Резервная копия не создана"
        log "Response: $response"
        return 1
    fi
}

# Функция для проверки целостности резервной копии
verify_backup() {
    local backup_file="$1"
    
    if [ ! -f "$backup_file" ]; then
        log "ОШИБКА: Файл резервной копии не найден: $backup_file"
        return 1
    fi
    
    # Проверяем, что файл не пустой
    local size=$(stat -f%z "$backup_file" 2>/dev/null || stat -c%s "$backup_file" 2>/dev/null)
    if [ "$size" -eq 0 ]; then
        log "ОШИБКА: Файл резервной копии пуст: $backup_file"
        return 1
    fi
    
    # Проверяем, что это валидный ZIP (если это ZIP)
    if [[ "$backup_file" == *.zip ]]; then
        if ! unzip -tq "$backup_file" 2>/dev/null; then
            log "ОШИБКА: ZIP архив поврежден: $backup_file"
            return 1
        fi
    fi
    
    log "Резервная копия проверена: $backup_file (размер: $size байт)"
    return 0
}

# Основная функция
main() {
    log "=== Начало резервного копирования ==="
    
    # Проверяем доступность сервера
    if ! curl -s -f "$BASE_URL/health" > /dev/null; then
        log "ОШИБКА: Сервер недоступен на $BASE_URL"
        exit 1
    fi
    
    # Создаем резервную копию
    if ! create_backup true true false "both"; then
        log "ОШИБКА: Не удалось создать резервную копию"
        exit 1
    fi
    
    # Находим последний созданный бэкап
    local latest_backup=$(ls -t "$BACKUP_DIR"/backup_*.zip 2>/dev/null | head -n 1)
    if [ -n "$latest_backup" ]; then
        verify_backup "$latest_backup"
    fi
    
    # Очищаем старые бэкапы
    cleanup_old_backups
    
    log "=== Резервное копирование завершено ==="
}

# Запуск
main "$@"

