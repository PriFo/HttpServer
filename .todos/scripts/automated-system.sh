#!/bin/bash

# Automated TODO System - Main daemon script
# Runs continuous scanning and reporting

set -e

PROJECT_DIR="${PROJECT_DIR:-.}"
SCAN_INTERVAL="${SCAN_INTERVAL:-3600}"  # 1 hour default
LOG_FILE="${PROJECT_DIR}/.todos/system.log"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

main_loop() {
    log "🚀 Automated TODO System started"
    log "Scan interval: ${SCAN_INTERVAL}s"
    
    while true; do
        log "🔄 Starting scan cycle..."
        
        # Run scan
        if bash "${PROJECT_DIR}/.todos/scripts/scan-todos.sh"; then
            log "✅ Scan completed successfully"
        else
            log "❌ Scan failed"
        fi
        
        # Generate report
        if node "${PROJECT_DIR}/.todos/scripts/generate-report.js"; then
            log "✅ Report generated"
        else
            log "❌ Report generation failed"
        fi
        
        # Auto-assign tasks (if Python available)
        if command -v python3 &> /dev/null; then
            if python3 "${PROJECT_DIR}/.todos/scripts/assign-tasks.py"; then
                log "✅ Tasks reassigned"
            fi
        fi
        
        log "⏳ Waiting ${SCAN_INTERVAL}s until next scan..."
        sleep "$SCAN_INTERVAL"
    done
}

# Handle signals
trap 'log "🛑 System stopped"; exit 0' SIGINT SIGTERM

# Run main loop
main_loop

