# Скрипт для диагностики 500 ошибок в backend
# Использование: .\diagnose-500-error.ps1

Write-Host "`n╔══════════════════════════════════════════════════════════╗" -ForegroundColor Red
Write-Host "║     🔍 ДИАГНОСТИКА 500 ОШИБОК В BACKEND                 ║" -ForegroundColor Red
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Red

Write-Host "`n📋 Инструкция по диагностике:" -ForegroundColor Cyan
Write-Host "`n1. Убедитесь, что backend запущен" -ForegroundColor Yellow
Write-Host "   Проверка:" -ForegroundColor White
try {
    $health = Invoke-RestMethod -Uri "http://localhost:9999/health" -Method GET -TimeoutSec 2
    Write-Host "   ✅ Backend работает" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Backend не запущен! Запустите: .\start-backend.ps1" -ForegroundColor Red
    exit 1
}

Write-Host "`n2. Откройте окно терминала с логами backend" -ForegroundColor Yellow
Write-Host "   В этом окне вы увидите логи в реальном времени" -ForegroundColor White

Write-Host "`n3. Выполните действие, которое вызывает ошибку 500" -ForegroundColor Yellow
Write-Host "   Например: загрузка базы данных через UI" -ForegroundColor White

Write-Host "`n4. Внимательно смотрите в логи backend в момент ошибки" -ForegroundColor Yellow
Write-Host "   Ищите следующие ключевые слова:" -ForegroundColor White
Write-Host "   • ERROR или FATAL" -ForegroundColor Red
Write-Host "   • panic:" -ForegroundColor Red
Write-Host "   • sql:" -ForegroundColor Red
Write-Host "   • runtime error:" -ForegroundColor Red
Write-Host "   • nil pointer" -ForegroundColor Red

Write-Host "`n5. Скопируйте строку ошибки и 5-10 строк до/после" -ForegroundColor Yellow
Write-Host "   Это поможет точно определить проблему" -ForegroundColor White

Write-Host "`n📝 Что искать в логах:" -ForegroundColor Cyan
Write-Host "`n   [handleUploadProjectDatabase] или [handleCreateProjectDatabase]" -ForegroundColor White
Write-Host "   Эти строки покажут, какой обработчик вызван" -ForegroundColor Gray

Write-Host "`n   ERROR: ..." -ForegroundColor Red
Write-Host "   Это основная ошибка" -ForegroundColor Gray

Write-Host "`n   panic: runtime error: ..." -ForegroundColor Red
Write-Host "   Это критическая ошибка, ниже будет stack trace" -ForegroundColor Gray

Write-Host "`n   sql: ..." -ForegroundColor Red
Write-Host "   Ошибка базы данных (например: no such table, database is locked)" -ForegroundColor Gray

Write-Host "`n🔧 Типичные проблемы:" -ForegroundColor Cyan
Write-Host "`n   1. serviceDB is nil" -ForegroundColor Yellow
Write-Host "      Решение: Проверьте инициализацию в server_init.go" -ForegroundColor White

Write-Host "`n   2. sql: database is closed" -ForegroundColor Yellow
Write-Host "      Решение: Проверьте, что БД не закрыта преждевременно" -ForegroundColor White

Write-Host "`n   3. nil pointer dereference" -ForegroundColor Yellow
Write-Host "      Решение: Добавьте проверки на nil" -ForegroundColor White

Write-Host "`n   4. FOREIGN KEY constraint failed" -ForegroundColor Yellow
Write-Host "      Решение: Убедитесь, что проект существует" -ForegroundColor White

Write-Host "`n💡 Совет:" -ForegroundColor Cyan
Write-Host "   Если ошибка не видна в логах, проверьте:" -ForegroundColor White
Write-Host "   • Middleware может перехватывать ошибки" -ForegroundColor Gray
Write-Host "   • Ошибка может быть в другом обработчике" -ForegroundColor Gray
Write-Host "   • Проверьте логи frontend (браузер DevTools)" -ForegroundColor Gray

Write-Host "`n📄 После нахождения ошибки:" -ForegroundColor Cyan
Write-Host "   Скопируйте точную строку ошибки и отправьте для исправления" -ForegroundColor White
Write-Host "`n"

