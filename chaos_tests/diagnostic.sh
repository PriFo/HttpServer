#!/bin/bash
# Диагностический скрипт для проверки окружения перед запуском Chaos Monkey тестов

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Счетчики
PASSED=0
FAILED=0
WARNINGS=0

# Функция для вывода результата проверки
check_result() {
    local status=$1
    local message=$2
    local fix_hint=$3
    
    if [ "$status" -eq 0 ]; then
        echo -e "${GREEN}[ OK ]${NC} $message"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}[FAIL]${NC} $message"
        if [ -n "$fix_hint" ]; then
            echo -e "${YELLOW}      → $fix_hint${NC}"
        fi
        ((FAILED++))
        return 1
    fi
}

# Функция для предупреждения
warn() {
    local message=$1
    echo -e "${YELLOW}[WARN]${NC} $message"
    ((WARNINGS++))
}

# Функция для информации
info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║         Chaos Monkey Diagnostic Tool                      ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# 1. Проверка окружения
echo "--- 1. Environment Check ---"

# Определение ОС
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    if grep -q Microsoft /proc/version 2>/dev/null || grep -q WSL /proc/version 2>/dev/null; then
        OS_TYPE="Linux (WSL)"
    else
        OS_TYPE="Linux"
    fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS_TYPE="macOS"
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    OS_TYPE="Windows (Git Bash/Cygwin)"
else
    OS_TYPE="Unknown ($OSTYPE)"
fi

info "OS: $OS_TYPE"

# Проверка shell
if [ -n "$BASH_VERSION" ]; then
    info "Shell: bash $BASH_VERSION"
    check_result 0 "Running in Bash"
else
    check_result 1 "Not running in Bash" "Please run this script with: bash diagnostic.sh"
    echo ""
    echo -e "${RED}Error: This script requires Bash.${NC}"
    echo "If you're on Windows, please use:"
    echo "  - WSL (Windows Subsystem for Linux)"
    echo "  - Git Bash"
    echo "  - Cygwin"
    exit 1
fi

echo ""

# 2. Проверка зависимостей
echo "--- 2. Dependencies Check ---"

# Проверка curl
if command -v curl &> /dev/null; then
    CURL_PATH=$(command -v curl)
    CURL_VERSION=$(curl --version | head -n1)
    check_result 0 "curl is found at $CURL_PATH"
    info "  Version: $CURL_VERSION"
else
    check_result 1 "curl is not installed" \
        "Install: Ubuntu/Debian: sudo apt-get install curl | macOS: brew install curl | Windows: Use WSL"
fi

# Проверка jq
if command -v jq &> /dev/null; then
    JQ_PATH=$(command -v jq)
    JQ_VERSION=$(jq --version 2>/dev/null || echo "unknown")
    check_result 0 "jq is found at $JQ_PATH"
    info "  Version: $JQ_VERSION"
else
    check_result 1 "jq is not installed" \
        "Install: Ubuntu/Debian: sudo apt-get install jq | macOS: brew install jq | Windows: Use WSL"
fi

# Проверка bc
if command -v bc &> /dev/null; then
    BC_PATH=$(command -v bc)
    check_result 0 "bc is found at $BC_PATH"
else
    check_result 1 "bc is not installed" \
        "Install: Ubuntu/Debian: sudo apt-get install bc | macOS: brew install bc | Windows: Use WSL"
fi

# Проверка ps (для мониторинга ресурсов)
if command -v ps &> /dev/null; then
    check_result 0 "ps is available"
else
    warn "ps is not available (needed for resource monitoring)"
fi

echo ""

# 3. Проверка бэкенда
echo "--- 3. Backend Status Check ---"

BASE_URL="${BASE_URL:-http://localhost:9999}"

# Проверка процесса бэкенда
BACKEND_PROCESS=""
if pgrep -f "httpserver_no_gui" > /dev/null 2>&1; then
    BACKEND_PROCESS="httpserver_no_gui"
    PID=$(pgrep -f "httpserver_no_gui" | head -1)
    check_result 0 "Backend process 'httpserver_no_gui' is running (PID: $PID)"
elif pgrep -f "httpserver" > /dev/null 2>&1; then
    BACKEND_PROCESS="httpserver"
    PID=$(pgrep -f "httpserver" | head -1)
    check_result 0 "Backend process 'httpserver' is running (PID: $PID)"
else
    check_result 1 "Backend process is not running" \
        "Start the backend server: ./httpserver_no_gui.exe or ./httpserver"
fi

# Проверка порта
if command -v netstat &> /dev/null; then
    if netstat -an 2>/dev/null | grep -q ":9999.*LISTEN" || netstat -an 2>/dev/null | grep -q "9999.*LISTENING"; then
        check_result 0 "Port 9999 is listening"
    else
        check_result 1 "Port 9999 is not in use" \
            "Check if backend is running and listening on port 9999"
    fi
elif command -v ss &> /dev/null; then
    if ss -an 2>/dev/null | grep -q ":9999.*LISTEN"; then
        check_result 0 "Port 9999 is listening"
    else
        check_result 1 "Port 9999 is not in use" \
            "Check if backend is running and listening on port 9999"
    fi
else
    warn "Cannot check port (netstat/ss not available)"
fi

# Проверка доступности API
info "Testing API endpoint: $BASE_URL/api/config"
HTTP_CODE=0
RESPONSE_BODY=""

if command -v curl &> /dev/null; then
    RESPONSE=$(curl -s -w "\n%{http_code}" --max-time 5 "$BASE_URL/api/config" 2>&1)
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    RESPONSE_BODY=$(echo "$RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" -eq 200 ]; then
        check_result 0 "GET /api/config returned HTTP 200"
        
        # Проверка валидности JSON
        if command -v jq &> /dev/null; then
            if echo "$RESPONSE_BODY" | jq . > /dev/null 2>&1; then
                check_result 0 "Response is valid JSON"
            else
                check_result 1 "Response is not valid JSON" \
                    "Backend may be returning an error or HTML instead of JSON"
            fi
        else
            warn "Cannot validate JSON (jq not installed)"
        fi
    elif [ "$HTTP_CODE" -eq 0 ]; then
        check_result 1 "GET /api/config failed: connection refused" \
            "Backend is not running or not accessible at $BASE_URL"
    elif [ "$HTTP_CODE" -ge 500 ]; then
        check_result 1 "GET /api/config returned HTTP $HTTP_CODE (server error)" \
            "Backend is running but encountering errors"
    elif [ "$HTTP_CODE" -ge 400 ]; then
        warn "GET /api/config returned HTTP $HTTP_CODE (client error)"
        info "  Response: $(echo "$RESPONSE_BODY" | head -c 200)"
    else
        check_result 1 "GET /api/config returned unexpected HTTP $HTTP_CODE"
    fi
else
    check_result 1 "Cannot test API (curl not installed)"
fi

echo ""

# 4. Проверка предусловий
echo "--- 4. Prerequisites Check ---"

# Проверка ARLIAI_API_KEY
if [ -n "$ARLIAI_API_KEY" ]; then
    check_result 0 "ARLIAI_API_KEY is set"
    info "  Key length: ${#ARLIAI_API_KEY} characters"
else
    warn "ARLIAI_API_KEY is not set (some AI tests may fail)"
    info "  Set it with: export ARLIAI_API_KEY='your-key'"
fi

# Проверка исполняемого файла бэкенда
BACKEND_EXECUTABLES=("httpserver_no_gui.exe" "httpserver_no_gui" "httpserver.exe" "httpserver" "./bin/httpserver_no_gui.exe" "../bin/httpserver_no_gui.exe")
BACKEND_FOUND=0

for exec_file in "${BACKEND_EXECUTABLES[@]}"; do
    if [ -f "$exec_file" ] && [ -x "$exec_file" ]; then
        check_result 0 "Executable found: $exec_file"
        BACKEND_FOUND=1
        info "  Size: $(du -h "$exec_file" | cut -f1)"
        break
    fi
done

if [ $BACKEND_FOUND -eq 0 ]; then
    warn "Backend executable not found in common locations"
    info "  Searched: ${BACKEND_EXECUTABLES[*]}"
    info "  Current directory: $(pwd)"
fi

# Проверка файла базы данных
DB_FILES=("1c_data.db" "data.db" "service.db" "./data/1c_data.db" "../data/1c_data.db")
DB_FOUND=0

for db_file in "${DB_FILES[@]}"; do
    if [ -f "$db_file" ] && [ -r "$db_file" ]; then
        check_result 0 "Database file is accessible: $db_file"
        DB_FOUND=1
        info "  Size: $(du -h "$db_file" | cut -f1)"
        break
    fi
done

if [ $DB_FOUND -eq 0 ]; then
    warn "Database file not found in common locations"
    info "  Searched: ${DB_FILES[*]}"
    info "  This may be normal if database is created dynamically"
fi

echo ""

# Итоги
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                      Summary                                ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo -e "${GREEN}Passed:${NC} $PASSED"
echo -e "${RED}Failed:${NC} $FAILED"
if [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}Warnings:${NC} $WARNINGS"
fi
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All critical checks passed!${NC}"
    echo "You can proceed with running chaos tests."
    exit 0
else
    echo -e "${RED}❌ Some checks failed.${NC}"
    echo "Please fix the issues above before running chaos tests."
    exit 1
fi

