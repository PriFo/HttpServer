# Быстрый старт

Краткое руководство для быстрого начала работы с проектом.

## Установка

### Backend

```bash
# Клонируйте репозиторий
git clone <repository-url>
cd HttpServer

# Установите зависимости
go mod download
go mod tidy
```

### Frontend

```bash
cd frontend
pnpm install
```

## Запуск

### Backend

```bash
# Запуск сервера
go run main.go
```

Сервер будет доступен на `http://localhost:9999`

### Frontend

```bash
cd frontend
pnpm run dev
```

Frontend будет доступен на `http://localhost:3000`

## Swagger UI

После запуска сервера:

1. Откройте `http://localhost:9999/swagger/index.html`
2. Просматривайте и тестируйте API эндпоинты

## Генерация Swagger документации

```bash
make swagger
```

## Docker

```bash
# Запуск всех сервисов
docker-compose up -d

# Просмотр логов
docker-compose logs -f

# Остановка
docker-compose down
```

## Следующие шаги

- Прочитайте [README.md](../README.md) для подробной документации
- Изучите [Swagger Usage Guide](./SWAGGER_USAGE.md) для работы со Swagger
- См. [Migration to Gin Guide](./MIGRATION_TO_GIN.md) для миграции handlers

