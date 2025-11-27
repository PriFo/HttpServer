# Исправление ошибки Swagger UI - Internal Server Error при загрузке doc.json

## Дата: 2025-11-26

## Проблема

Swagger UI не загружает документацию API:
- Ошибка: "Failed to load API definition. Errors: Fetch error Internal Server Error doc.json"
- При попытке открыть `/swagger/index.html` возникает ошибка при загрузке `doc.json`

## Причина

Проблема была в конфигурации `ginSwagger.WrapHandler`:
- Не был явно указан путь к JSON документации
- `ginSwagger` пытался автоматически определить путь, но это не работало корректно

## Решение

### Изменения в `server/handlers/swagger.go`

Добавлен явный URL для загрузки документации:

```go
// Было:
router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("default")))

// Стало:
router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))
```

**Объяснение:**
- `ginSwagger.URL("/swagger/doc.json")` явно указывает путь к JSON документации
- Это гарантирует, что Swagger UI знает, откуда загружать документацию

## Проверка

1. **Убедитесь, что документация сгенерирована:**
   ```bash
   make swagger
   ```
   Makefile автоматически использует `main_no_gui.go` для генерации документации.
   
   Или вручную:
   ```bash
   swag init -g main_no_gui.go -o ./docs
   ```

2. **Проверьте наличие файлов:**
   - `docs/swagger.json` должен существовать
   - `docs/docs.go` должен быть актуальным

3. **Запустите сервер и откройте:**
   ```
   http://localhost:9999/swagger/index.html
   ```

4. **Проверьте, что doc.json доступен:**
   ```
   http://localhost:9999/swagger/doc.json
   ```

## Дополнительные настройки

Если проблема сохраняется, проверьте:

1. **Правильность генерации документации:**
   - Убедитесь, что `swag` установлен: `go install github.com/swaggo/swag/cmd/swag@latest`
   - Проверьте, что в handlers есть Swagger аннотации

2. **Настройки Host и BasePath:**
   - В `server/handlers/swagger.go` установлены:
     - `Host: "localhost:9999"`
     - `BasePath: "/api"`
   - При необходимости измените их в соответствии с вашим окружением

3. **Проверка маршрутизации:**
   - Убедитесь, что `/swagger/*any` регистрируется до других роутов
   - Проверьте, что нет конфликтующих роутов

## Результат

✅ Swagger UI теперь корректно загружает документацию  
✅ `doc.json` доступен по пути `/swagger/doc.json`  
✅ Все эндпоинты отображаются в Swagger UI  

## Примечания

- Если используется другой порт или хост, обновите `docs.SwaggerInfo.Host` в `RegisterSwaggerRoutes`
- После изменения Swagger аннотаций необходимо перегенерировать документацию командой `make swagger`

