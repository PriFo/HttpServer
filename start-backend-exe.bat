@echo off
chcp 65001 >nul
echo ========================================
echo Запуск бэкенда на порту 9999
echo ========================================
echo.

REM Переходим в директорию скрипта
cd /d "%~dp0"

REM Проверяем, не запущен ли уже backend
netstat -ano | findstr :9999 >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo [ПРЕДУПРЕЖДЕНИЕ] Порт 9999 уже занят!
    echo Возможно, backend уже запущен.
    echo.
    choice /C YN /M "Продолжить запуск"
    if errorlevel 2 exit /b 0
)

REM Проверяем наличие скомпилированного exe
if exist "httpserver_no_gui.exe" (
    echo [ИНФО] Используется скомпилированный exe файл
    echo.
    REM Устанавливаем API ключ для ArliAI
    set ARLIAI_API_KEY=597dbe7e-16ca-4803-ab17-5fa084909f37
    
    REM Запускаем бэкенд в фоне
    echo Запуск сервера с AI нормализацией...
    start "Backend Server" /MIN "" "%~dp0httpserver_no_gui.exe"
    
    REM Ждем запуска
    timeout /t 3 /nobreak >nul
    
    REM Проверяем, что сервер запустился
    curl -m 3 http://localhost:9999/health >nul 2>&1
    if %ERRORLEVEL% EQU 0 (
        echo [УСПЕХ] Backend сервер запущен и отвечает на порту 9999
        echo.
        echo API доступно по адресу: http://localhost:9999
        echo Health check: http://localhost:9999/health
    ) else (
        echo [ОШИБКА] Backend сервер не отвечает
        echo Проверьте логи в окне сервера
    )
) else (
    echo [ИНФО] Скомпилированный exe не найден, используем go run
    echo.
    
    REM Проверяем наличие Go
    where go >nul 2>&1
    if %ERRORLEVEL% NEQ 0 (
        echo [ОШИБКА] Go не найден в PATH
        echo Установите Go и добавьте его в PATH
        pause
        exit /b 1
    )
    
    REM Устанавливаем API ключ для ArliAI
    set ARLIAI_API_KEY=597dbe7e-16ca-4803-ab17-5fa084909f37
    
    REM Запускаем бэкенд
    echo Запуск сервера с AI нормализацией...
    go run -tags no_gui main_no_gui.go
)

echo.
pause

