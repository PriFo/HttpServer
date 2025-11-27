# Расширенные возможности Chaos Monkey

## Новые инструменты

### 1. config.py - Централизованная конфигурация

Управление всеми настройками из одного места.

**Использование:**
```python
from config import get_config

config = get_config()
print(config.base_url)
print(config.concurrent_requests)
```

**Конфигурационный файл:**
```json
{
  "base_url": "http://localhost:9999",
  "tests": {
    "concurrent_config": {
      "num_requests": 10
    }
  }
}
```

### 2. notifier.py - Система уведомлений

Отправка уведомлений о результатах тестов.

**Типы уведомлений:**
- **Email** - отправка на email
- **File** - сохранение в файл
- **Console** - вывод в консоль

**Использование:**
```python
from notifier import create_notifier, send_test_results

# Email уведомления
email_notifier = create_notifier('email', {
    'smtp_server': 'smtp.gmail.com',
    'username': 'user@gmail.com',
    'password': 'password',
    'to_emails': ['admin@example.com']
})

# File уведомления
file_notifier = create_notifier('file', {
    'notifications_file': './notifications.log'
})

# Отправка результатов
results = {'concurrent_config': True, 'ai_failure': False}
send_test_results(results, [email_notifier, file_notifier])
```

### 3. scheduler.py - Планировщик тестов

Запуск тестов по расписанию.

**Добавление задачи:**
```bash
python scheduler.py --add "daily_tests" \
  --cron "daily" \
  --command run_complete_suite.py --auto-start
```

**Запуск по времени:**
```bash
python scheduler.py --add "morning_tests" \
  --cron "09:00" \
  --command chaos_monkey.py --test all
```

**Запуск каждые N минут:**
```bash
python scheduler.py --add "hourly_tests" \
  --cron "every 60 minutes" \
  --command run_quick_test.py
```

**Список задач:**
```bash
python scheduler.py --list
```

**Запуск в режиме демона:**
```bash
python scheduler.py --daemon
```

### 4. health_check.py - Проверка здоровья системы

Проверка состояния системы перед запуском тестов.

**Использование:**
```bash
# Базовая проверка
python health_check.py

# JSON вывод
python health_check.py --json
```

**Проверяет:**
- Доступность сервера (/health)
- Доступность API (/api/config)
- Наличие процесса сервера
- Свободное место на диске
- Установленные зависимости

## Интеграция

### Использование конфигурации в тестах

```python
from config import get_config

config = get_config()

# Использование настроек
base_url = config.base_url
num_requests = config.concurrent_requests
```

### Автоматические уведомления

```python
from notifier import create_notifier, send_test_results
from chaos_monkey import main as run_tests

# Запуск тестов
results = run_tests()

# Отправка уведомлений
notifiers = [
    create_notifier('console'),
    create_notifier('file', {'notifications_file': './notifications.log'})
]
send_test_results(results, notifiers)
```

### Health check перед тестами

```python
from health_check import HealthChecker

checker = HealthChecker()
health = checker.run_all_checks()

if health['all_passed']:
    # Запуск тестов
    run_tests()
else:
    print("Система не готова к тестированию")
```

## Примеры использования

### Ежедневный запуск тестов

```bash
# Добавить в cron или Task Scheduler
python scheduler.py --add "daily" \
  --cron "daily" \
  --command run_complete_suite.py --auto-start
```

### Мониторинг с уведомлениями

```python
#!/usr/bin/env python3
from health_check import HealthChecker
from notifier import create_notifier, send_test_results
from run_complete_suite import main as run_tests

# Проверка здоровья
checker = HealthChecker()
health = checker.run_all_checks()

if health['all_passed']:
    # Запуск тестов
    results = run_tests()
    
    # Уведомления
    notifiers = [
        create_notifier('file'),
        create_notifier('console')
    ]
    send_test_results(results, notifiers)
else:
    print("Система не готова")
```

### Конфигурируемый запуск

```python
#!/usr/bin/env python3
from config import get_config
from chaos_monkey import main as run_tests

config = get_config('custom_config.json')

# Использование настроек из конфига
import os
os.environ['CHAOS_BASE_URL'] = config.base_url
os.environ['CHAOS_CONCURRENT_REQUESTS'] = str(config.concurrent_requests)

run_tests()
```

## Конфигурационные файлы

### chaos_config.json

```json
{
  "base_url": "http://localhost:9999",
  "reports_dir": "./reports",
  "tests": {
    "concurrent_config": {
      "num_requests": 20
    }
  },
  "server": {
    "auto_start": true,
    "api_key": "your-api-key"
  }
}
```

### chaos_schedule.json

```json
{
  "daily_tests": {
    "cron": "daily",
    "command": ["run_complete_suite.py", "--auto-start"],
    "enabled": true
  },
  "hourly_quick": {
    "cron": "every 60 minutes",
    "command": ["chaos_monkey.py", "--test", "concurrent_config", "--quick"],
    "enabled": true
  }
}
```

## Переменные окружения

Все настройки можно переопределить через переменные окружения:

```bash
export CHAOS_BASE_URL="http://localhost:9999"
export CHAOS_CONCURRENT_REQUESTS=20
export CHAOS_AUTO_START=true
export ARLIAI_API_KEY="your-key"
```

## Рекомендации

1. **Используйте конфигурационные файлы** для разных окружений
2. **Настройте уведомления** для мониторинга результатов
3. **Используйте планировщик** для регулярного тестирования
4. **Проверяйте здоровье системы** перед запуском тестов
5. **Сохраняйте историю** через file notifier

