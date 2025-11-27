#!/bin/bash

# Скрипт для восстановления из резервной копии

set -e

# Конфигурация
BACKUP_DIR="${BACKUP_DIR:-./data/backups}"
RESTORE_DIR="${RESTORE_DIR:-./data/restored}"
LOG_FILE="${LOG_FILE:-./logs/restore.log}"

# Создаем директории
mkdir -p "$RESTORE_DIR"
mkdir -p "$(dirname "$LOG_FILE")"

# Функция для логирования
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Функция для выбора резервной копии
select_backup() {
    local backups=($(ls -t "$BACKUP_DIR"/backup_*.zip 2>/dev/null))
    
    if [ ${#backups[@]} -eq 0 ]; then
        log "ОШИБКА: Резервные копии не найдены в $BACKUP_DIR"
        exit 1
    fi
    
    if [ -n "$1" ]; then
        # Используем указанный файл
        local selected="$1"
        if [ ! -f "$selected" ]; then
            log "ОШИБКА: Файл не найден: $selected"
            exit 1
        fi
        echo "$selected"
    else
        # Используем последний бэкап
        echo "${backups[0]}"
    fi
}

# Функция для восстановления из ZIP
restore_from_zip() {
    local backup_file="$1"
    local restore_path="$2"
    
    log "Восстановление из: $backup_file"
    log "Восстановление в: $restore_path"
    
    # Проверяем целостность архива
    if ! unzip -tq "$backup_file" 2>/dev/null; then
        log "ОШИБКА: ZIP архив поврежден: $backup_file"
        return 1
    fi
    
    # Создаем директорию для восстановления
    mkdir -p "$restore_path"
    
    # Распаковываем архив
    log "Распаковка архива..."
    if unzip -q "$backup_file" -d "$restore_path"; then
        log "Архив успешно распакован"
        return 0
    else
        log "ОШИБКА: Не удалось распаковать архив"
        return 1
    fi
}

# Функция для восстановления из директории
restore_from_directory() {
    local backup_dir="$1"
    local restore_path="$2"
    
    log "Восстановление из директории: $backup_dir"
    log "Восстановление в: $restore_path"
    
    if [ ! -d "$backup_dir" ]; then
        log "ОШИБКА: Директория не найдена: $backup_dir"
        return 1
    fi
    
    # Копируем файлы
    log "Копирование файлов..."
    if cp -r "$backup_dir"/* "$restore_path"/ 2>/dev/null; then
        log "Файлы успешно скопированы"
        return 0
    else
        log "ОШИБКА: Не удалось скопировать файлы"
        return 1
    fi
}

# Основная функция
main() {
    local backup_file_or_dir="$1"
    
    log "=== Начало восстановления ==="
    
    # Выбираем резервную копию
    local backup_path
    if [ -n "$backup_file_or_dir" ]; then
        backup_path="$backup_file_or_dir"
    else
        backup_path=$(select_backup)
    fi
    
    log "Выбрана резервная копия: $backup_path"
    
    # Определяем тип бэкапа и восстанавливаем
    if [[ "$backup_path" == *.zip ]]; then
        if ! restore_from_zip "$backup_path" "$RESTORE_DIR"; then
            log "ОШИБКА: Восстановление не удалось"
            exit 1
        fi
    elif [ -d "$backup_path" ]; then
        if ! restore_from_directory "$backup_path" "$RESTORE_DIR"; then
            log "ОШИБКА: Восстановление не удалось"
            exit 1
        fi
    else
        log "ОШИБКА: Неизвестный тип резервной копии: $backup_path"
        exit 1
    fi
    
    log "=== Восстановление завершено ==="
    log "Восстановленные файлы находятся в: $RESTORE_DIR"
    log ""
    log "ВАЖНО: Проверьте восстановленные файлы перед использованием!"
    log "ВАЖНО: Убедитесь, что сервер остановлен перед заменой файлов!"
}

# Запуск
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Использование: $0 [путь_к_резервной_копии]"
    echo ""
    echo "Примеры:"
    echo "  $0                                    # Восстановить из последнего бэкапа"
    echo "  $0 ./data/backups/backup_20231123.zip # Восстановить из указанного файла"
    exit 0
fi

main "$@"

