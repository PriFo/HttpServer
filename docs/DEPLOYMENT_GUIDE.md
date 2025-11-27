# Руководство по развертыванию

**Версия:** 1.0  
**Дата:** 2025-11-23

---

## 📋 Содержание

1. [Предварительные требования](#предварительные-требования)
2. [Переменные окружения](#переменные-окружения)
3. [Развертывание с Docker](#развертывание-с-docker)
4. [Развертывание без Docker](#развертывание-без-docker)
5. [Проверка развертывания](#проверка-развертывания)
6. [Мониторинг](#мониторинг)
7. [Откат изменений](#откат-изменений)

---

## Предварительные требования

### Системные требования

- **ОС:** Linux (Ubuntu 20.04+), Windows Server 2019+, или macOS
- **CPU:** Минимум 2 ядра, рекомендуется 4+
- **RAM:** Минимум 4GB, рекомендуется 8GB+
- **Диск:** Минимум 20GB свободного места

### Программное обеспечение

- **Docker:** 20.10+ (для Docker развертывания)
- **Docker Compose:** 2.0+ (для Docker развертывания)
- **Go:** 1.21+ (для развертывания без Docker)
- **Node.js:** 20+ (для развертывания без Docker)
- **SQLite:** 3.35+ (встроен в большинство систем)

---

## Переменные окружения

### Backend переменные

Создайте файл `.env` в корне проекта:

```bash
# Server
SERVER_PORT=9999
SERVER_HOST=0.0.0.0

# Databases
DATABASE_PATH=/app/data/1c_data.db
NORMALIZED_DATABASE_PATH=/app/data/normalized_data.db
SERVICE_DATABASE_PATH=/app/data/service.db

# API Keys
ARLIAI_API_KEY=your_arliai_api_key_here
ARLIAI_MODEL=GLM-4.5-Air
OPENROUTER_API_KEY=your_openrouter_api_key_here

# Database Connection Pool
MAX_OPEN_CONNS=25
MAX_IDLE_CONNS=5
CONN_MAX_LIFETIME=300s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Security
JWT_SECRET=your_jwt_secret_here
API_KEY=your_api_key_here
```

### Frontend переменные

Создайте файл `frontend/.env.local`:

```bash
# Backend URL (для production используйте реальный URL)
NEXT_PUBLIC_BACKEND_URL=http://localhost:9999

# Environment
NODE_ENV=production

# Analytics (опционально)
NEXT_PUBLIC_ANALYTICS_ID=your_analytics_id
```

---

## Развертывание с Docker

### Быстрый старт

1. **Клонируйте репозиторий:**
```bash
git clone <repository-url>
cd HttpServer
```

2. **Настройте переменные окружения:**
```bash
cp .env.example .env
# Отредактируйте .env файл
```

3. **Запустите с Docker Compose:**
```bash
docker-compose up -d
```

4. **Проверьте статус:**
```bash
docker-compose ps
```

### Производственное развертывание

1. **Используйте production docker-compose:**
```bash
docker-compose -f docker-compose.prod.yml up -d
```

2. **Настройте reverse proxy (Nginx):**
```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    location /api {
        proxy_pass http://localhost:9999;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

3. **Настройте SSL (Let's Encrypt):**
```bash
certbot --nginx -d your-domain.com
```

---

## Развертывание без Docker

### Backend

1. **Соберите приложение:**
```bash
go build -o httpserver main_no_gui.go
```

2. **Создайте директории для данных:**
```bash
mkdir -p data
```

3. **Запустите сервер:**
```bash
./httpserver
```

### Frontend

1. **Установите зависимости:**
```bash
cd frontend
npm ci
```

2. **Соберите приложение:**
```bash
npm run build
```

3. **Запустите production сервер:**
```bash
npm start
```

---

## Проверка развертывания

### Health Check

```bash
# Backend
curl http://localhost:9999/health

# Frontend
curl http://localhost:3000
```

### Проверка API

```bash
# Получить список баз данных
curl http://localhost:9999/api/databases

# Получить статистику системы
curl http://localhost:9999/api/system/summary
```

### Проверка логов

```bash
# Docker
docker-compose logs -f backend
docker-compose logs -f frontend

# Без Docker
tail -f logs/server.log
```

---

## Мониторинг

### Метрики

- **Health endpoint:** `http://localhost:9999/health`
- **Metrics endpoint:** `http://localhost:9999/api/monitoring/metrics`
- **System summary:** `http://localhost:9999/api/system/summary`

### Логирование

Логи сохраняются в:
- **Docker:** `docker-compose logs`
- **Без Docker:** `logs/server.log`

### Алерты

Настройте мониторинг для следующих метрик:
- CPU использование > 80%
- RAM использование > 80%
- Disk использование > 90%
- Response time > 1s
- Error rate > 1%

---

## Откат изменений

### Docker

1. **Остановите текущие контейнеры:**
```bash
docker-compose down
```

2. **Вернитесь к предыдущей версии:**
```bash
git checkout <previous-commit>
docker-compose up -d
```

3. **Или используйте предыдущий образ:**
```bash
docker-compose pull
docker-compose up -d --force-recreate
```

### Без Docker

1. **Остановите сервер:**
```bash
pkill httpserver
```

2. **Вернитесь к предыдущей версии:**
```bash
git checkout <previous-commit>
go build -o httpserver main_no_gui.go
./httpserver
```

### Восстановление базы данных

```bash
# Создайте резервную копию перед обновлением
cp data/service.db data/service.db.backup

# Восстановите из резервной копии
cp data/service.db.backup data/service.db
```

---

## Troubleshooting

### Проблемы с портами

Если порт занят:
```bash
# Проверьте, что использует порт
lsof -i :9999  # Linux/macOS
netstat -ano | findstr :9999  # Windows

# Измените порт в .env или docker-compose.yml
```

### Проблемы с базой данных

```bash
# Проверьте права доступа
ls -la data/

# Проверьте целостность БД
sqlite3 data/service.db "PRAGMA integrity_check;"
```

### Проблемы с памятью

```bash
# Увеличьте лимиты в docker-compose.yml
deploy:
  resources:
    limits:
      memory: 4G
```

---

## Дополнительные ресурсы

- [Docker документация](https://docs.docker.com/)
- [Nginx документация](https://nginx.org/en/docs/)
- [Let's Encrypt документация](https://letsencrypt.org/docs/)

---

*Последнее обновление: 2025-11-23*

