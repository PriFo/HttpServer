# Исправление ошибки "Unexpected end of multipart data" при загрузке баз данных

**Дата:** 26 ноября 2025  
**Проблема:** Ошибка "Unexpected end of multipart data" при попытке загрузить базу данных через веб-интерфейс

---

## Проблема

При попытке загрузить базу данных (.db файл) через веб-интерфейс возникала ошибка:
```
Error: Unexpected end of multipart data
```

**Причина:**
- Frontend отправляет multipart/form-data запрос в Next.js API route
- Next.js API route пересоздавал FormData, читая файлы в память
- При пересоздании FormData терялся boundary или возникала проблема с сериализацией File объектов
- Backend получал неполные или поврежденные multipart данные

---

## Решение

Изменен подход к проксированию multipart/form-data запросов в `frontend/app/api/clients/[clientId]/projects/[projectId]/databases/route.ts`:

### Было:
- Чтение FormData из запроса
- Пересоздание FormData для передачи на backend
- Конвертация File объектов в Blob/ArrayBuffer
- Потеря boundary и ошибка "Unexpected end of multipart data"

### Стало:
- **Потоковая передача** тела запроса напрямую от клиента к backend
- Сохранение оригинального Content-Type с boundary
- Передача request.body как ReadableStream
- Отсутствие промежуточной обработки и потери данных

---

## Изменения в коде

### `frontend/app/api/clients/[clientId]/projects/[projectId]/databases/route.ts`

**Основные изменения:**
1. Удалено пересоздание FormData
2. Удалена обработка File объектов
3. Добавлена потоковая передача через `request.body`
4. Сохранение оригинального Content-Type с boundary

**Ключевой код:**
```typescript
// Передаем тело запроса напрямую с сохранением всех заголовков
const response = await fetch(`${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases`, {
  method: 'POST',
  body: request.body, // Потоковая передача без обработки
  signal: controller.signal,
  headers: {
    'Content-Type': contentType, // Сохраняем оригинальный Content-Type с boundary
    'Content-Length': contentLength || '',
    'X-Request-ID': requestID,
  },
  duplex: 'half' as any, // Для потоковой передачи в Node.js 18+
} as RequestInit)
```

---

## Преимущества нового подхода

✅ **Сохраняет boundary** - оригинальный Content-Type передается без изменений  
✅ **Нет потери данных** - тело запроса передается напрямую, без промежуточной обработки  
✅ **Эффективность** - потоковая передача не требует загрузки всего файла в память  
✅ **Совместимость** - работает с любыми размерами файлов (до лимита backend: 500MB)

---

## Проверка

Для проверки исправления:

1. Откройте веб-интерфейс: `http://localhost:3000`
2. Перейдите: **Проекты** → **AITAS-MDM-2025-001** → **Базы данных**
3. Нажмите: **"Добавить базу данных"**
4. Выберите файл .db и загрузите

**Ожидаемый результат:** Файл загружается без ошибок "Unexpected end of multipart data"

---

## Технические детали

### Поддержка потоковой передачи в Node.js

В Node.js 18+ fetch API поддерживает потоковую передачу через параметр `duplex: 'half'`:

```typescript
duplex: 'half' as any  // Поддержка потоковой передачи request body
```

### Ограничения

- Требуется Node.js 18+ для поддержки потоковой передачи
- Максимальный размер файла: 500MB (ограничение backend)
- Таймаут: 10 минут (обычные файлы), 15 минут (файлы > 100MB)

---

## Связанные файлы

- `frontend/app/api/clients/[clientId]/projects/[projectId]/databases/route.ts` - исправленный файл
- `server/client_legacy_handlers.go` - обработчик загрузки на backend
- `server/server_start_shutdown.go` - регистрация роутов

---

**Статус:** ✅ Исправлено  
**Тестирование:** Требуется проверка на реальной загрузке файла

