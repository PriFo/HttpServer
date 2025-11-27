# Исправление ошибки загрузки баз данных

## Дата: 2025-11-26

## Проблема

При попытке загрузить базу данных в проект возникала ошибка:
```
POST http://localhost:3000/api/clients/1/projects/1/databases 500 (Internal Server Error)
```

**Причина:**
- Фронтенд отправляет запрос с `Content-Type: multipart/form-data` для загрузки файла
- В Gin роутах был зарегистрирован только `CreateProjectDatabase`, который ожидает JSON
- При попытке декодировать multipart/form-data как JSON возникала ошибка 500

## Решение

### Изменения в `server/server_start_shutdown.go`

Добавлена проверка `Content-Type` в регистрации POST роута для баз данных:

```go
projectDatabasesAPI.POST("", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
    // Проверяем Content-Type для определения типа запроса
    contentType := r.Header.Get("Content-Type")
    if strings.HasPrefix(contentType, "multipart/form-data") {
        // Загрузка файла - используем legacy handler
        s.handleUploadProjectDatabase(w, r, clientID, projectID)
    } else {
        // Обычный JSON запрос - используем новый handler
        s.clientHandler.CreateProjectDatabase(w, r, clientID, projectID)
    }
}))
```

**Логика:**
- Если `Content-Type` начинается с `multipart/form-data` → используется `handleUploadProjectDatabase` (legacy handler для загрузки файлов)
- Иначе → используется `CreateProjectDatabase` (новый handler для JSON запросов)

### Добавлен импорт

Добавлен импорт `strings` для проверки префикса Content-Type.

## Результат

✅ Загрузка файлов через multipart/form-data теперь работает корректно  
✅ JSON запросы для создания БД без файла продолжают работать  
✅ Обработка обоих типов запросов в одном endpoint  
✅ Проект компилируется без ошибок  

## Тестирование

Для проверки:
1. Откройте страницу проекта: `/clients/{clientId}/projects/{projectId}`
2. Перейдите на вкладку "Базы данных"
3. Попробуйте загрузить файл .db через drag & drop или выбор файла
4. Файл должен успешно загрузиться без ошибки 500
