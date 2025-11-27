# Быстрое решение проблемы подключения к Backend

## Проблема

Ошибка в браузере:
```
Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999.
```

## Быстрое решение (3 шага)

### Шаг 1: Проверьте статус backend
```batch
check-backend-status.bat
```

Если backend не запущен, переходите к шагу 2.

### Шаг 2: Запустите backend
```batch
start-backend-exe.bat
```

Или через go run:
```batch
start-backend.bat
```

### Шаг 3: Проверьте работу
Откройте браузер: http://localhost:3000

Dashboard должен загрузиться без ошибок.

## Альтернатива: Запуск обоих серверов

Если нужно запустить и backend, и frontend:
```batch
start-all.bat
```

## Проверка системы

Для полной проверки состояния:
```powershell
powershell -ExecutionPolicy Bypass -File .\quick-check.ps1
```

## Автоматический мониторинг

Для автоматического мониторинга и перезапуска:
```powershell
powershell -ExecutionPolicy Bypass -File .\monitor-backend.ps1 -AutoRestart
```

## Что было исправлено

1. ✅ Исправлена логика API запросов в `frontend/lib/api-utils.ts`
2. ✅ Улучшена обработка ошибок во всех API routes
3. ✅ Добавлены скрипты для удобного управления backend

## Дополнительная информация

Подробная документация:
- `BACKEND_CONNECTION_FIX.md` - Полное описание проблемы и решения
- `UTILITIES_README.md` - Документация по всем утилитам
- `COMPLETE_SOLUTION_SUMMARY.md` - Полная сводка решения

