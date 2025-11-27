@echo off
chcp 65001 >nul
echo ========================================
echo Проверка статуса Backend сервера
echo ========================================
echo.

REM Проверяем, занят ли порт 9999
netstat -ano | findstr :9999 >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo [СТАТУС] Порт 9999 занят
    echo.
    echo Проверка доступности сервера...
    curl -m 3 http://localhost:9999/health 2>nul
    if %ERRORLEVEL% EQU 0 (
        echo.
        echo [УСПЕХ] Backend сервер работает и отвечает на запросы
        echo.
        echo API доступно по адресу: http://localhost:9999
        echo Health check: http://localhost:9999/health
    ) else (
        echo.
        echo [ОШИБКА] Порт занят, но сервер не отвечает на запросы
        echo Возможно, сервер еще запускается или произошла ошибка
    )
) else (
    echo [СТАТУС] Порт 9999 свободен
    echo [ОШИБКА] Backend сервер не запущен
    echo.
    echo Для запуска используйте:
    echo   start-backend-exe.bat  - запуск через exe (быстрее)
    echo   start-backend.bat      - запуск через go run
)

echo.
pause

