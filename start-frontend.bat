@echo off
chcp 65001 >nul
echo Запуск фронтенда на порту 3000...
echo.

REM Переходим в директорию скрипта
cd /d "%~dp0"

REM Проверяем наличие Node.js
where npm >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Ошибка: Node.js не найден в PATH
    echo Установите Node.js и добавьте его в PATH
    pause
    exit /b 1
)

REM Переходим в директорию фронтенда
cd frontend

REM Запускаем фронтенд
echo Запуск Next.js...
npm run dev

pause

