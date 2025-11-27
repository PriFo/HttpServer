# Руководство по использованию Chaos Monkey тестов

## Быстрый старт

### 1. Установка зависимостей

```bash
cd chaos_tests
pip install -r requirements.txt
```

### 2. Запуск тестов (рекомендуемый способ)

**С автоматическим запуском сервера:**
```bash
python test_runner.py --test all --auto-start
```

**С ожиданием готовности сервера:**
```bash
python test_runner.py --test all --wait-timeout 120
```

**Быстрый режим (меньше запросов):**
```bash
python test_runner.py --test concurrent_config --quick
```

### 3. Запуск отдельных тестов

```bash
# Конкурентные обновления конфигурации
python test_runner.py --test concurrent_config

# Нормализация с невалидными данными
python test_runner.py --test invalid_normalization

# Устойчивость к сбоям AI
python test_runner.py --test ai_failure

# Работа с большими объемами
python test_runner.py --test large_data
```

## Улучшенные тесты

### DatabaseLockTest - Тест блокировок БД

Проверяет поведение системы при конкурентных операциях с базой данных.

```bash
python improved_tests.py --test db_lock
```

**Что проверяется:**
- Блокировки БД при параллельных обновлениях
- Пропуски версий в истории конфигурации
- Обработка ошибок "database is locked"

### StressTest - Стресс-тест endpoints

Нагрузочное тестирование API endpoints.

```bash
python improved_tests.py --test stress
```

**Что проверяется:**
- Время ответа при высокой нагрузке
- Обработка таймаутов
- Статус-коды ответов
- Пропускная способность (requests/second)

### ResourceMonitor - Мониторинг ресурсов

Мониторинг использования CPU и памяти процесса.

```bash
python improved_tests.py --test monitor
```

**Что проверяется:**
- Использование CPU (%)
- Использование памяти (MB)
- Тренды использования ресурсов
- Возможные утечки памяти

## Прямой запуск chaos_monkey.py

Если сервер уже запущен:

```bash
# Все тесты
python chaos_monkey.py --test all

# Отдельный тест
python chaos_monkey.py --test concurrent_config

# Быстрый режим
python chaos_monkey.py --test concurrent_config --quick
```

## Диагностика

### Проверка сервера

```powershell
# PowerShell
.\diagnose_server.ps1

# Или вручную
curl http://localhost:9999/health
curl http://localhost:9999/api/config
```

### Запуск сервера вручную

```powershell
cd E:\HttpServer
$env:ARLIAI_API_KEY="597dbe7e-16ca-4803-ab17-5fa084909f37"
.\httpserver_no_gui.exe
```

## Параметры командной строки

### test_runner.py

```
--test {all,concurrent_config,invalid_normalization,ai_failure,large_data}
    Тест для запуска (по умолчанию: all)

--base-url URL
    Базовый URL сервера (по умолчанию: http://localhost:9999)

--quick
    Быстрый режим (меньше запросов)

--auto-start
    Автоматически запустить сервер, если он не запущен

--wait-timeout SECONDS
    Таймаут ожидания сервера (по умолчанию: 60)
```

### chaos_monkey.py

```
--test {all,concurrent_config,invalid_normalization,ai_failure,large_data}
    Тест для запуска

--base-url URL
    Базовый URL сервера

--quick
    Быстрый режим

--report-dir DIR
    Директория для отчетов (по умолчанию: ./reports)
```

## Отчеты

Все отчеты сохраняются в директории `reports/`:

- `concurrent_config_YYYYMMDD_HHMMSS.md` - отчет по конкурентным обновлениям
- `invalid_normalization_YYYYMMDD_HHMMSS.md` - отчет по невалидным данным
- `ai_failure_YYYYMMDD_HHMMSS.md` - отчет по сбоям AI
- `large_data_YYYYMMDD_HHMMSS.md` - отчет по большим объемам
- `chaos_test_summary_YYYYMMDD_HHMMSS.md` - сводный отчет

## Логи

Логи сохраняются в `logs/chaos_monkey_YYYYMMDD.log`

## Примеры использования

### Полный цикл тестирования

```bash
# 1. Установка зависимостей
pip install -r requirements.txt

# 2. Запуск всех тестов с автоматическим стартом сервера
python test_runner.py --test all --auto-start

# 3. Просмотр отчетов
ls reports/
```

### Тестирование конкретной проблемы

```bash
# Тест блокировок БД
python improved_tests.py --test db_lock

# Стресс-тест endpoint конфигурации
python improved_tests.py --test stress

# Мониторинг ресурсов во время нормализации
python improved_tests.py --test monitor
```

### Быстрая проверка

```bash
# Быстрый тест конкурентных обновлений
python test_runner.py --test concurrent_config --quick

# Проверка подключения
python test_connection.py
```

## Устранение проблем

### Сервер не запускается

1. Проверьте наличие `httpserver_no_gui.exe`
2. Установите `ARLIAI_API_KEY` в переменные окружения
3. Проверьте, что порт 9999 свободен
4. Запустите сервер в видимом режиме для просмотра ошибок

### Тесты не запускаются

1. Убедитесь, что сервер запущен и отвечает:
   ```bash
   curl http://localhost:9999/health
   ```

2. Проверьте зависимости:
   ```bash
   pip install -r requirements.txt
   ```

3. Используйте `--auto-start` для автоматического запуска сервера

### Ошибки подключения

1. Проверьте URL сервера: `--base-url http://localhost:9999`
2. Проверьте файрвол и сетевые настройки
3. Убедитесь, что сервер слушает на правильном порту

## Дополнительная информация

- Полная документация: `README.md`
- Быстрый старт: `QUICKSTART.md`
- Продолжение разработки: `DEVELOPMENT_CONTINUED.md`
- Анализ сборки: `BUILD_ANALYSIS.md`

