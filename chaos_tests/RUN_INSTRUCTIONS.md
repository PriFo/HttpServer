# Инструкции по запуску Chaos Monkey тестов

## ⚠️ Важно: Перед запуском тестов

**Сервер должен быть запущен и доступен на `http://localhost:9999`**

### Проверка доступности сервера

```bash
# Windows (PowerShell)
curl http://localhost:9999/api/config

# WSL/Linux
curl http://localhost:9999/api/config
```

Если сервер не запущен, запустите его:
```bash
# Запуск сервера (пример)
./httpserver_no_gui.exe
# или
./httpserver
```

## Запуск тестов

### Вариант 1: Python (рекомендуется)

```bash
cd chaos_tests

# Установка зависимостей (один раз)
python3 -m pip install requests psutil

# Проверка окружения
python3 chaos_monkey.py --test all

# Запуск всех тестов
python3 chaos_monkey.py --test all

# Запуск отдельного теста
python3 chaos_monkey.py --test concurrent_config
python3 chaos_monkey.py --test invalid_normalization
python3 chaos_monkey.py --test ai_failure
python3 chaos_monkey.py --test large_data
```

### Вариант 2: Bash (требует WSL/Git Bash)

```bash
cd chaos_tests

# Проверка окружения
bash diagnostic.sh

# Запуск всех тестов
bash run_all_tests.sh

# Запуск отдельного теста
bash test_concurrent_config.sh
```

## Результаты

После выполнения тестов:

1. **Отчеты** находятся в `chaos_tests/reports/`
2. **Логи** находятся в `chaos_tests/logs/`

## Устранение проблем

### Ошибка: "Connection refused"
- Убедитесь, что сервер запущен
- Проверьте, что сервер слушает порт 9999
- Проверьте файрвол

### Ошибка: "psutil not installed"
- Тесты будут работать, но мониторинг ресурсов будет недоступен
- Установите: `pip install psutil`

### Ошибка: "requests not installed"
- Установите: `pip install requests`

## Примечания

- Тесты могут временно изменить конфигурацию (автоматически восстанавливается)
- Некоторые тесты требуют наличия клиентов и проектов в БД
- Тест `large_data` может занять до 5 минут

