@echo off
chcp 65001 >nul
echo ========================================
echo Запуск бэкенда в режиме отладки
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

REM Устанавливаем API ключ для ArliAI
set ARLIAI_API_KEY=597dbe7e-16ca-4803-ab17-5fa084909f37

REM Создаем папку для логов, если ее нет
if not exist "logs" mkdir logs

REM Получаем timestamp для имени лог-файла
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set datetime=%%I
set timestamp=%datetime:~0,8%-%datetime:~8,6%
set logfile=logs\backend-%timestamp%.log

echo [ИНФО] Лог-файл: %logfile%
echo.

REM Проверяем наличие скомпилированного exe
if exist "httpserver_no_gui.exe" (
    echo [ИНФО] Используется скомпилированный exe файл
    echo.
    echo Запуск сервера с AI нормализацией...
    echo Все логи будут сохранены в: %logfile%
    echo.
    
    REM Запускаем бэкенд в отдельном окне (БЕЗ /MIN, чтобы видеть ошибки)
    REM Логирование будет в отдельном окне, файл логов будет создан, если доступна команда tee
    start "Backend Server (Debug)" cmd /k "title Backend Server Debug && echo Запуск сервера... && echo Логи также сохраняются в: %logfile% && echo. && (httpserver_no_gui.exe 2>&1 & echo Сервер остановлен. Нажмите любую клавишу для закрытия.) && pause"
    
    REM Альтернативный способ с логированием в файл (если tee недоступен)
    REM start "Backend Server (Debug)" cmd /k "title Backend Server Debug && echo Запуск сервера... && httpserver_no_gui.exe > %logfile% 2>&1"
    
    REM Ждем запуска
    timeout /t 5 /nobreak >nul
    
    REM Проверяем, что сервер запустился
    curl -m 3 http://localhost:9999/health >nul 2>&1
    if %ERRORLEVEL% EQU 0 (
        echo [УСПЕХ] Backend сервер запущен и отвечает на порту 9999
        echo.
        echo API доступно по адресу: http://localhost:9999
        echo Health check: http://localhost:9999/health
        echo Логи: %logfile%
        echo.
        echo Окно сервера открыто. Закройте его для остановки сервера.
    ) else (
        echo [ОШИБКА] Backend сервер не отвечает после 5 секунд
        echo Проверьте окно сервера и логи в файле: %logfile%
        echo.
        echo Возможные причины:
        echo   - Ошибка инициализации базы данных
        echo   - Проблема с конфигурацией
        echo   - Ошибка создания контейнера зависимостей
        echo.
        echo Для детальной диагностики запустите:
        echo   diagnose-backend-startup.ps1
    )
) else (
    echo [ИНФО] Скомпилированный exe не найден, используем go run
    echo.
    echo Запуск через go run (все логи будут видны в консоли)...
    echo.
    
    REM Запускаем через go run (логи будут в консоли)
    REM Для сохранения в файл используйте: go run -tags no_gui main_no_gui.go > %logfile% 2>&1
    go run -tags no_gui main_no_gui.go
)

pause

