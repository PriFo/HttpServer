# Pre-commit hook для проверки критических TODO (Windows PowerShell)
# Использование: Copy-Item .todos\scripts\pre-commit-check.ps1 .git\hooks\pre-commit

$ErrorActionPreference = "Stop"

$PROJECT_ROOT = git rev-parse --show-toplevel
$TODO_DB = Join-Path $PROJECT_ROOT ".todos\tasks.json"

if (-not (Test-Path $TODO_DB)) {
    exit 0
}

try {
    $data = Get-Content $TODO_DB | ConvertFrom-Json
    $criticalCount = ($data.tasks | Where-Object { 
        $_.priority -eq "CRITICAL" -and $_.status -eq "OPEN" 
    }).Count
    
    if ($criticalCount -gt 0) {
        Write-Host "🚨 Найдено $criticalCount критических TODO. Пожалуйста, исправьте перед коммитом." -ForegroundColor Red
        Write-Host "Используйте 'git commit --no-verify' чтобы обойти проверку." -ForegroundColor Yellow
        exit 1
    }
} catch {
    # Если ошибка при чтении, пропускаем проверку
    exit 0
}

exit 0


