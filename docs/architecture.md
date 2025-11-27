# Архитектура (Enterprise рефакторинг)

## Слои
```
cmd/server/main.go
        │
        ▼
internal/container.Container ──► internal/api/routes ──► server/handlers (Gin)
                                              │
                                              └── legacy_routes (fallback)
```

## Legacy-хендлеры
- `server/handlers/legacy/` — HTTP слой (package `legacy`).
- `server/services/legacy/` — временные сервисные адаптеры.
- `internal/api/routes/legacy_routes.go` + `server/legacy_routes_adapter.go` — централизованная регистрация `/api/legacy`.

## Автоматизация
- `scripts/migrate_server_file.sh` — копирование `server_*.go` → legacy.
- `scripts/server_refactor_status.sh` — метрики `server.go`.
- `.github/workflows/refactoring-checks.yml` — контроль размеров и остатка legacy-файлов.

## Контрольный список
1. Миграция обработчиков группами (Benchmarks → Similarity → Export/Groups → Classification → Complex).
2. После каждой группы — `go build ./...`, `scripts/server_refactor_status.sh`, `npm run todos:scan`.
3. Обновление `docs/MIGRATION.md` и smoke-тестов (`tests/regression/smoke_test.go`).

