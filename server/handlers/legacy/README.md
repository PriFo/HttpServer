# Legacy handlers

Структура для постепенной миграции `server_*.go`:

```
server/
  handlers/
    legacy/
      benchmarks_legacy.go
      similarity/
        similarity_api_legacy.go
      classification/
        classification_management_legacy.go
      ...
  services/
    legacy/
      classification_management_legacy.go
```

*Все новые файлы используют `package legacy` и регистрируются через `internal/api/routes/legacy_routes.go`.*

