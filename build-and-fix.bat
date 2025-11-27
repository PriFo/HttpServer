@echo off
REM Батник для автоматической сборки проекта и генерации плана исправления ошибок
REM Для Windows пользователей, которые предпочитают .bat файлы

echo === Запуск системы генерации плана исправления ошибок ===
echo.

REM Проверяем наличие PowerShell
where pwsh >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo Используется PowerShell Core (pwsh)
    pwsh -ExecutionPolicy Bypass -File "build-and-fix.ps1"
    goto :end
)

where powershell >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo Используется Windows PowerShell
    powershell -ExecutionPolicy Bypass -File "build-and-fix.ps1"
    goto :end
)

echo Ошибка: PowerShell не найден!
echo Установите PowerShell и попробуйте снова.
pause
exit /b 1

:end
if %ERRORLEVEL% NEQ 0 (
    echo.
    echo Ошибка при выполнении скрипта!
    pause
    exit /b %ERRORLEVEL%
)

