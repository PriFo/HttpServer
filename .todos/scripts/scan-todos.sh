#!/bin/bash

# Automated TODO Scanner
# Сканирует код на наличие TODO, FIXME, HACK комментариев

set -e

PROJECT_DIR="${1:-.}"
TODO_DB=".todos/tasks.json"
CONFIG_FILE=".todos/config.json"
TEAM_CONFIG=".todos/team.json"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔄 Starting automated TODO scan...${NC}"

# Инициализация БД если нужно
if [[ ! -f "$TODO_DB" ]]; then
    mkdir -p "$(dirname "$TODO_DB")"
    echo '{"tasks": [], "metadata": {"lastScan": null, "totalScans": 0, "version": "1.0.0"}}' > "$TODO_DB"
fi

# Функция классификации приоритета
classify_priority() {
    local line=$1
    local upper_line=$(echo "$line" | tr '[:lower:]' '[:upper:]')
    
    if [[ $upper_line =~ CRITICAL|PANIC|NOT\ IMPLEMENTED|FIXME|HACK ]]; then
        echo "CRITICAL"
    elif [[ $upper_line =~ HIGH|IMPORTANT|BUG|ERROR ]]; then
        echo "HIGH"
    elif [[ $upper_line =~ MEDIUM|OPTIMIZE|REFACTOR ]]; then
        echo "MEDIUM"
    else
        echo "LOW"
    fi
}

# Функция определения типа задачи
classify_type() {
    local line=$1
    local upper_line=$(echo "$line" | tr '[:lower:]' '[:upper:]')
    
    if [[ $upper_line =~ FIXME ]]; then
        echo "FIXME"
    elif [[ $upper_line =~ HACK ]]; then
        echo "HACK"
    elif [[ $upper_line =~ REFACTOR ]]; then
        echo "REFACTOR"
    else
        echo "TODO"
    fi
}

# Функция определения команды на основе типа файла
determine_team() {
    local file=$1
    local ext="${file##*.}"
    
    case $ext in
        go|py|java|rb)
            echo "backend-team"
            ;;
        ts|tsx|js|jsx|vue|svelte)
            echo "frontend-team"
            ;;
        sh|yml|yaml|Dockerfile|dockerfile)
            echo "devops"
            ;;
        *)
            echo "unassigned"
            ;;
    esac
}

# Функция генерации ID задачи
generate_id() {
    local file=$1
    local line=$2
    echo "$(echo -n "$file:$line" | sha256sum | cut -d' ' -f1 | cut -c1-16)"
}

# Функция извлечения описания из строки
extract_description() {
    local line=$1
    # Убираем комментарии и лишние пробелы
    echo "$line" | sed -e 's/^[[:space:]]*\/\/[[:space:]]*//' \
                       -e 's/^[[:space:]]*#*[[:space:]]*//' \
                       -e 's/^[[:space:]]*\*[[:space:]]*//' \
                       -e 's/TODO[[:space:]]*(.*)[[:space:]]*://' \
                       -e 's/TODO[[:space:]]*://' \
                       -e 's/FIXME[[:space:]]*://' \
                       -e 's/HACK[[:space:]]*://' \
                       -e 's/REFACTOR[[:space:]]*://' \
                       | sed 's/^[[:space:]]*//;s/[[:space:]]*$//'
}

# Функция извлечения TODO из файла
extract_todos_from_file() {
    local file=$1
    local line_num=0
    local found_count=0
    
    # Пропускаем файлы из exclude patterns
    if [[ "$file" =~ node_modules|vendor|\.git|\.min\.|dist/|build/ ]]; then
        return
    fi
    
    while IFS= read -r line || [ -n "$line" ]; do
        ((line_num++))
        
        # Поиск TODO
        if [[ $line =~ TODO|FIXME|HACK|REFACTOR ]]; then
            local priority=$(classify_priority "$line")
            local type=$(classify_type "$line")
            local description=$(extract_description "$line")
            local team=$(determine_team "$file")
            local task_id=$(generate_id "$file" "$line_num")
            
            # Проверяем, не существует ли уже задача
            if ! jq -e ".tasks[] | select(.id == \"$task_id\")" "$TODO_DB" > /dev/null 2>&1; then
                # Создаем новую задачу
                local task_json=$(jq -n \
                    --arg id "$task_id" \
                    --arg file "$file" \
                    --argjson line "$line_num" \
                    --arg description "$description" \
                    --arg type "$type" \
                    --arg priority "$priority" \
                    --arg team "$team" \
                    --arg created "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
                    '{
                        "id": $id,
                        "file": $file,
                        "line": $line,
                        "description": $description,
                        "type": $type,
                        "priority": $priority,
                        "status": "OPEN",
                        "assignedTo": $team,
                        "createdAt": $created,
                        "updatedAt": $created,
                        "estimatedHours": 0,
                        "dependencies": [],
                        "relatedFiles": []
                    }')
                
                # Добавляем задачу в БД
                jq ".tasks += [$task_json]" "$TODO_DB" > "$TODO_DB.tmp" && mv "$TODO_DB.tmp" "$TODO_DB"
                
                echo -e "${GREEN}📝 Created task: $task_id${NC} ($priority) - $file:$line_num"
                ((found_count++))
            fi
        fi
    done < "$file"
    
    if [ $found_count -gt 0 ]; then
        echo -e "${BLUE}   Found $found_count new task(s) in $file${NC}"
    fi
}

# Основная функция сканирования
scan_directory() {
    local dir=$1
    local total_files=0
    local total_tasks=0
    
    echo -e "${YELLOW}Scanning directory: $dir${NC}"
    
    # Сканирование Go файлов
    while IFS= read -r -d '' file; do
        extract_todos_from_file "$file"
        ((total_files++))
    done < <(find "$dir" -type f -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" -print0 2>/dev/null)
    
    # Сканирование TypeScript/JavaScript файлов
    while IFS= read -r -d '' file; do
        extract_todos_from_file "$file"
        ((total_files++))
    done < <(find "$dir" -type f \( -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" \) -not -path "*/node_modules/*" -not -path "*/.git/*" -print0 2>/dev/null)
    
    # Сканирование Python файлов
    while IFS= read -r -d '' file; do
        extract_todos_from_file "$file"
        ((total_files++))
    done < <(find "$dir" -type f -name "*.py" -not -path "*/venv/*" -not -path "*/.git/*" -print0 2>/dev/null)
    
    # Сканирование Shell скриптов
    while IFS= read -r -d '' file; do
        extract_todos_from_file "$file"
        ((total_files++))
    done < <(find "$dir" -type f \( -name "*.sh" -o -name "*.bash" \) -not -path "*/.git/*" -print0 2>/dev/null)
    
    # Сканирование Markdown файлов
    while IFS= read -r -d '' file; do
        extract_todos_from_file "$file"
        ((total_files++))
    done < <(find "$dir" -type f -name "*.md" -not -path "*/.git/*" -print0 2>/dev/null)
    
    echo -e "${GREEN}✅ Scanned $total_files files${NC}"
}

# Обновление метаданных
update_metadata() {
    local scan_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local total_scans=$(jq '.metadata.totalScans // 0' "$TODO_DB")
    ((total_scans++))
    
    jq ".metadata.lastScan = \"$scan_time\" | .metadata.totalScans = $total_scans" "$TODO_DB" > "$TODO_DB.tmp" && mv "$TODO_DB.tmp" "$TODO_DB"
}

# Главная функция
main() {
    if [ ! -f "$TODO_DB" ]; then
        echo -e "${RED}Error: TODO database not found at $TODO_DB${NC}"
        exit 1
    fi
    
    scan_directory "$PROJECT_DIR"
    update_metadata
    
    local total_tasks=$(jq '.tasks | length' "$TODO_DB")
    local open_tasks=$(jq '[.tasks[] | select(.status == "OPEN")] | length' "$TODO_DB")
    local critical_tasks=$(jq '[.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN")] | length' "$TODO_DB")
    
    echo -e "${GREEN}✅ Scan completed at $(date)${NC}"
    echo -e "${BLUE}📊 Statistics:${NC}"
    echo -e "   Total tasks: $total_tasks"
    echo -e "   Open tasks: $open_tasks"
    echo -e "   ${RED}Critical tasks: $critical_tasks${NC}"
}

main "$@"
