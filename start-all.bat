@echo off
chcp 65001 >nul
echo ========================================
echo Запуск системы нормализации данных
echo ========================================
echo.

REM Проверяем наличие Go
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [ОШИБКА] Go не найден в PATH
    echo Установите Go и добавьте его в PATH
    pause
    exit /b 1
)

REM Проверяем наличие Node.js
where npm >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [ОШИБКА] Node.js не найден в PATH
    echo Установите Node.js и добавьте его в PATH
    pause
    exit /b 1
)

REM Переходим в директорию скрипта
cd /d "%~dp0"

echo [1/2] Запуск бэкенда на порту 9999...
start "Backend Server" cmd /k "cd /d %~dp0 && go run main_no_gui.go"
timeout /t 3 /nobreak >nul

echo [2/2] Запуск фронтенда на порту 3000...
start "Frontend Server" cmd /k "cd /d %~dp0frontend && npm run dev"

echo.
echo ========================================
echo Серверы запущены!
echo.
echo Бэкенд:  http://localhost:9999
echo Фронтенд: http://localhost:3000
echo.
echo Для остановки закройте окна серверов
echo ========================================
echo.
pause

