# PowerShell скрипт для сборки Docker контейнера
# Использование: .\docker-build.ps1

param(
    [switch]$Build,
    [switch]$Run,
    [switch]$Stop,
    [switch]$Logs,
    [switch]$Clean,
    [string]$Service = "all"
)

Write-Host "`n╔══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║     🐳 DOCKER BUILD SCRIPT                               ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan

# Функция для проверки Docker
function Test-Docker {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Host "`n❌ Docker не установлен!" -ForegroundColor Red
        Write-Host "Установите Docker Desktop: https://www.docker.com/products/docker-desktop" -ForegroundColor Yellow
        exit 1
    }
    
    if (-not (Get-Command docker-compose -ErrorAction SilentlyContinue)) {
        Write-Host "`n❌ Docker Compose не установлен!" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "`n✅ Docker и Docker Compose установлены" -ForegroundColor Green
}

# Функция для создания директории data
function Initialize-DataDirectory {
    if (-not (Test-Path "data")) {
        Write-Host "`n📁 Создание директории data..." -ForegroundColor Yellow
        New-Item -ItemType Directory -Path "data" | Out-Null
        New-Item -ItemType Directory -Path "data/uploads" | Out-Null
        New-Item -ItemType Directory -Path "data/backups" | Out-Null
        New-Item -ItemType Directory -Path "data/temp" | Out-Null
        Write-Host "✅ Директория data создана" -ForegroundColor Green
    } else {
        Write-Host "`n✅ Директория data уже существует" -ForegroundColor Green
    }
}

# Функция для сборки
function Build-Docker {
    Write-Host "`n🔨 Сборка Docker контейнеров..." -ForegroundColor Yellow
    
    if ($Service -eq "backend" -or $Service -eq "all") {
        Write-Host "`n📦 Сборка Backend..." -ForegroundColor Cyan
        docker build -t httpserver-backend .
        if ($LASTEXITCODE -ne 0) {
            Write-Host "`n❌ Ошибка сборки Backend" -ForegroundColor Red
            exit 1
        }
        Write-Host "✅ Backend собран успешно" -ForegroundColor Green
    }
    
    if ($Service -eq "frontend" -or $Service -eq "all") {
        Write-Host "`n📦 Сборка Frontend..." -ForegroundColor Cyan
        docker build -t httpserver-frontend -f frontend/Dockerfile frontend/
        if ($LASTEXITCODE -ne 0) {
            Write-Host "`n❌ Ошибка сборки Frontend" -ForegroundColor Red
            exit 1
        }
        Write-Host "✅ Frontend собран успешно" -ForegroundColor Green
    }
    
    if ($Service -eq "all") {
        Write-Host "`n📦 Сборка через docker-compose..." -ForegroundColor Cyan
        docker-compose build
        if ($LASTEXITCODE -ne 0) {
            Write-Host "`n❌ Ошибка сборки через docker-compose" -ForegroundColor Red
            exit 1
        }
        Write-Host "✅ Все сервисы собраны успешно" -ForegroundColor Green
    }
}

# Функция для запуска
function Start-Docker {
    Write-Host "`n🚀 Запуск Docker контейнеров..." -ForegroundColor Yellow
    
    Initialize-DataDirectory
    
    docker-compose up -d
    if ($LASTEXITCODE -ne 0) {
        Write-Host "`n❌ Ошибка запуска контейнеров" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "`n✅ Контейнеры запущены" -ForegroundColor Green
    Write-Host "`n📊 Статус контейнеров:" -ForegroundColor Cyan
    docker-compose ps
    
    Write-Host "`n🌐 Доступные сервисы:" -ForegroundColor Cyan
    Write-Host "  • Backend:  http://localhost:9999" -ForegroundColor White
    Write-Host "  • Frontend: http://localhost:3000" -ForegroundColor White
    Write-Host "  • Health:   http://localhost:9999/health" -ForegroundColor White
}

# Функция для остановки
function Stop-Docker {
    Write-Host "`n🛑 Остановка Docker контейнеров..." -ForegroundColor Yellow
    docker-compose down
    Write-Host "✅ Контейнеры остановлены" -ForegroundColor Green
}

# Функция для просмотра логов
function Show-Logs {
    Write-Host "`n📋 Логи Docker контейнеров:" -ForegroundColor Yellow
    docker-compose logs -f
}

# Функция для очистки
function Clean-Docker {
    Write-Host "`n🧹 Очистка Docker ресурсов..." -ForegroundColor Yellow
    
    $confirm = Read-Host "Удалить все контейнеры, образы и volumes? (y/N)"
    if ($confirm -eq "y" -or $confirm -eq "Y") {
        docker-compose down -v --rmi all
        Write-Host "✅ Очистка завершена" -ForegroundColor Green
    } else {
        Write-Host "❌ Очистка отменена" -ForegroundColor Yellow
    }
}

# Основная логика
Test-Docker

if ($Build) {
    Build-Docker
}

if ($Run) {
    Start-Docker
}

if ($Stop) {
    Stop-Docker
}

if ($Logs) {
    Show-Logs
}

if ($Clean) {
    Clean-Docker
}

# Если параметры не указаны, показываем справку
if (-not ($Build -or $Run -or $Stop -or $Logs -or $Clean)) {
    Write-Host "`n📖 Использование:" -ForegroundColor Cyan
    Write-Host "  .\docker-build.ps1 -Build          # Собрать контейнеры" -ForegroundColor White
    Write-Host "  .\docker-build.ps1 -Run            # Запустить контейнеры" -ForegroundColor White
    Write-Host "  .\docker-build.ps1 -Build -Run     # Собрать и запустить" -ForegroundColor White
    Write-Host "  .\docker-build.ps1 -Stop           # Остановить контейнеры" -ForegroundColor White
    Write-Host "  .\docker-build.ps1 -Logs           # Показать логи" -ForegroundColor White
    Write-Host "  .\docker-build.ps1 -Clean           # Очистить все" -ForegroundColor White
    Write-Host "`n  Примеры:" -ForegroundColor Yellow
    Write-Host "  .\docker-build.ps1 -Build -Run     # Полная сборка и запуск" -ForegroundColor Green
    Write-Host "  .\docker-build.ps1 -Service backend # Собрать только backend" -ForegroundColor Green
}

