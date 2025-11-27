# Руководство по сборке Docker контейнера

**Дата:** 2025-11-26  
**Статус:** ✅ **ГОТОВО К ИСПОЛЬЗОВАНИЮ**

---

## 🐳 Структура Docker файлов

### Backend (Go)
- **Dockerfile** - многоэтапная сборка Go приложения без GUI
- Использует `main_no_gui.go` с build tag `no_gui`
- Финальный образ на базе `alpine:latest`

### Frontend (Next.js)
- **frontend/Dockerfile** - многоэтапная сборка Next.js приложения
- Финальный образ на базе `node:20-alpine`

### Docker Compose
- **docker-compose.yml** - оркестрация backend и frontend
- Настроены volumes для сохранения данных
- Healthcheck для backend

---

## 🚀 Быстрый старт

### 1. Сборка и запуск всех сервисов

```bash
# Сборка и запуск backend + frontend
docker-compose up --build -d

# Просмотр логов
docker-compose logs -f

# Остановка
docker-compose down
```

### 2. Только Backend

```bash
# Сборка backend
docker build -t httpserver-backend .

# Запуск backend
docker run -d \
  --name httpserver-backend \
  -p 9999:9999 \
  -v $(pwd)/data:/app/data \
  -e SERVER_PORT=9999 \
  -e ARLIAI_API_KEY=your_key_here \
  httpserver-backend
```

### 3. Только Frontend

```bash
# Сборка frontend
cd frontend
docker build -t httpserver-frontend .

# Запуск frontend
docker run -d \
  --name httpserver-frontend \
  -p 3000:3000 \
  -e BACKEND_URL=http://localhost:9999 \
  httpserver-frontend
```

---

## 📋 Переменные окружения

### Backend

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `SERVER_PORT` | Порт сервера | `9999` |
| `DATABASE_PATH` | Путь к основной БД | `/app/data/1c_data.db` |
| `NORMALIZED_DATABASE_PATH` | Путь к нормализованной БД | `/app/data/normalized_data.db` |
| `SERVICE_DATABASE_PATH` | Путь к сервисной БД | `/app/data/service.db` |
| `ARLIAI_API_KEY` | API ключ для AI нормализации | (пусто) |
| `ARLIAI_MODEL` | Модель AI | `GLM-4.5-Air` |
| `MAX_OPEN_CONNS` | Максимум открытых соединений | `25` |
| `MAX_IDLE_CONNS` | Максимум неактивных соединений | `5` |
| `CONN_MAX_LIFETIME` | Время жизни соединения | `300s` |

### Frontend

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `BACKEND_URL` | URL backend сервера | `http://backend:9999` |
| `NODE_ENV` | Режим Node.js | `production` |

---

## 📁 Volumes (монтируемые директории)

### Backend

- `./data:/app/data` - директория с базами данных и загруженными файлами
- `./1c_data.db:/app/1c_data.db:ro` - основная БД (read-only)
- `./normalized_data.db:/app/normalized_data.db:ro` - нормализованная БД (read-only)
- `./service.db:/app/service.db:ro` - сервисная БД (read-only)
- `./1c_processing:/app/1c_processing:ro` - файлы для генерации XML
- `./КПВЭД.txt:/app/КПВЭД.txt:ro` - файл классификатора

---

## 🔧 Сборка

### Backend

```bash
# Сборка с кэшем
docker build -t httpserver-backend .

# Сборка без кэша (чистая сборка)
docker build --no-cache -t httpserver-backend .

# Сборка с указанием Dockerfile
docker build -f Dockerfile -t httpserver-backend .
```

### Frontend

```bash
cd frontend
docker build -t httpserver-frontend .
```

---

## 🐛 Отладка

### Просмотр логов

```bash
# Все сервисы
docker-compose logs -f

# Только backend
docker-compose logs -f backend

# Только frontend
docker-compose logs -f frontend
```

### Вход в контейнер

```bash
# Backend
docker exec -it httpserver-backend sh

# Frontend
docker exec -it httpserver-frontend sh
```

### Проверка здоровья

```bash
# Backend healthcheck
docker exec httpserver-backend wget -q -O- http://localhost:9999/health

# Или через curl
docker exec httpserver-backend curl -f http://localhost:9999/health
```

---

## 📊 Мониторинг

### Статус контейнеров

```bash
docker-compose ps
```

### Использование ресурсов

```bash
docker stats
```

### Проверка портов

```bash
# Backend (9999)
curl http://localhost:9999/health

# Frontend (3000)
curl http://localhost:3000
```

---

## 🔄 Обновление

### Пересборка после изменений

```bash
# Остановка
docker-compose down

# Пересборка и запуск
docker-compose up --build -d
```

### Обновление только backend

```bash
docker-compose build backend
docker-compose up -d backend
```

### Обновление только frontend

```bash
docker-compose build frontend
docker-compose up -d frontend
```

---

## 🗑️ Очистка

### Удаление контейнеров

```bash
docker-compose down
```

### Удаление образов

```bash
docker-compose down --rmi all
```

### Полная очистка (контейнеры + образы + volumes)

```bash
docker-compose down -v --rmi all
```

### Очистка неиспользуемых ресурсов Docker

```bash
docker system prune -a
```

---

## ⚠️ Важные замечания

### 1. Базы данных

- Базы данных сохраняются в `./data` директории
- При первом запуске создаются пустые БД, если их нет
- Для переноса данных скопируйте `.db` файлы в `./data`

### 2. Порты

- Backend: `9999`
- Frontend: `3000`
- Убедитесь, что порты свободны перед запуском

### 3. Переменные окружения

- `ARLIAI_API_KEY` должен быть установлен для работы AI нормализации
- Можно создать `.env` файл для удобства:

```env
ARLIAI_API_KEY=your_key_here
ARLIAI_MODEL=GLM-4.5-Air
SERVER_PORT=9999
```

### 4. Права доступа

- Контейнеры запускаются от непривилегированных пользователей
- Убедитесь, что `./data` директория имеет правильные права

---

## 📝 Примеры использования

### Разработка

```bash
# Запуск в режиме разработки с hot reload (если настроено)
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up
```

### Production

```bash
# Запуск в production режиме
docker-compose up -d

# С логированием
docker-compose up -d && docker-compose logs -f
```

### Тестирование

```bash
# Запуск тестов в контейнере
docker-compose exec backend go test ./...
```

---

## 🔍 Troubleshooting

### Проблема: Backend не запускается

1. Проверьте логи: `docker-compose logs backend`
2. Проверьте, что порт 9999 свободен
3. Проверьте переменные окружения
4. Проверьте права на директорию `./data`

### Проблема: Frontend не подключается к Backend

1. Убедитесь, что `BACKEND_URL` правильный
2. Проверьте, что backend запущен: `docker-compose ps`
3. Проверьте сеть Docker: `docker network ls`

### Проблема: Базы данных не сохраняются

1. Проверьте, что volume `./data:/app/data` смонтирован
2. Проверьте права на директорию `./data`
3. Проверьте логи на ошибки доступа к БД

---

## 📄 Связанные файлы

- `Dockerfile` - сборка backend
- `frontend/Dockerfile` - сборка frontend
- `docker-compose.yml` - оркестрация
- `.dockerignore` - исключения для сборки

---

**Готово к использованию!** ✅

