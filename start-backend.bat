@echo off
chcp 65001 >nul
echo Запуск бэкенда на порту 9999...
echo.

REM Переходим в директорию скрипта
cd /d "%~dp0"

REM Проверяем наличие Go
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Ошибка: Go не найден в PATH
    echo Установите Go и добавьте его в PATH
    pause
    exit /b 1
)

REM Устанавливаем API ключ для ArliAI
set ARLIAI_API_KEY=597dbe7e-16ca-4803-ab17-5fa084909f37

REM Запускаем бэкенд
echo Запуск сервера с AI нормализацией...
go run -tags no_gui main_no_gui.go

pause

