# Server.go reduction roadmap

| Этап | Цель | KPI | Статус |
| --- | --- | --- | --- |
| 1. Вынести обработчики (`handle*`) | оставить в `server.go` только DI и маршруты | `server.go` ≤ 4000 строк; 0 функций `handle*` | В процессе (6809 строк, 52 `handle*`) |
| 2. Вынести middleware/утилиты | `server.go` ≤ 2000 строк | middleware в `server/middleware`, utils в `server/utils` | Заблокировано этапом 1 |
| 3. Минималистичный Server | `server.go` < 1000 строк | оставить `Server`, `NewServerWithConfig`, `setupRouter`, `Start`, `Shutdown` | Планируется |

## Метрики
- 24.11.2025: `wc -l server/server.go` → 6809 строк.
- Команда `scripts/server_refactor_status.sh` (todo) будет генерировать отчёт для CI.

## Действия
1. Создать поддиректории `server/handlers/legacy/{benchmarks,similarity,classification}`.
2. Использовать `scripts/migrate_server_file.sh` для переноса `server/server_benchmarks.go` (pilot).
3. Для новой директории `server/middleware/` перенести `Gin*` middleware (они уже выделены, нужно обновить импорты).
4. После каждого переноса обновлять `docs/MIGRATION.md` и запускать smoke-тесты.

