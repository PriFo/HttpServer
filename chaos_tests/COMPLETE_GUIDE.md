# Полное руководство по Chaos Monkey тестированию

## Обзор

Chaos Monkey - это комплексная система нагрузочного и нестандартного тестирования бэкенда, предназначенная для выявления слабых мест после рефакторинга.

## Быстрый старт

### 1. Установка зависимостей

```bash
cd chaos_tests
pip install -r requirements.txt
```

### 2. Запуск полного набора тестов

```bash
# Автоматический запуск сервера и всех тестов
python run_complete_suite.py --auto-start

# Быстрый режим
python run_complete_suite.py --auto-start --quick
```

### 3. Просмотр результатов

Откройте `reports/dashboard.html` в браузере для просмотра интерактивного дашборда.

## Структура тестов

### Базовые тесты

1. **concurrent_config** - Конкурентные обновления конфигурации
2. **invalid_normalization** - Нормализация с невалидными данными
3. **ai_failure** - Устойчивость к сбоям AI сервиса
4. **large_data** - Работа с большими объемами данных

### Улучшенные тесты

5. **db_lock** - Тест блокировок базы данных
6. **stress** - Стресс-тестирование endpoints
7. **monitor** - Мониторинг ресурсов

## Инструменты

### Основные скрипты

- `chaos_monkey.py` - Основные тесты
- `integrated_chaos_monkey.py` - Интегрированная версия всех тестов
- `improved_tests.py` - Улучшенные тесты
- `test_runner.py` - Улучшенный запуск с автоматическим стартом сервера
- `run_complete_suite.py` - Полный набор тестов с анализом и визуализацией

### Утилиты

- `report_analyzer.py` - Анализ истории отчетов
- `visualize_results.py` - Визуализация результатов
- `diagnose_server.ps1` - Диагностика сервера
- `test_connection.py` - Проверка подключения

## Использование

### Запуск отдельных тестов

```bash
# Базовые тесты
python chaos_monkey.py --test concurrent_config
python chaos_monkey.py --test invalid_normalization
python chaos_monkey.py --test ai_failure
python chaos_monkey.py --test large_data

# Улучшенные тесты
python improved_tests.py --test db_lock
python improved_tests.py --test stress
python improved_tests.py --test monitor

# Интегрированная версия
python integrated_chaos_monkey.py --test all
```

### Анализ результатов

```bash
# Анализ отчетов
python report_analyzer.py

# Визуализация
python visualize_results.py

# Полный анализ с визуализацией
python run_complete_suite.py --auto-start
```

### Автоматический запуск сервера

```bash
# С автоматическим запуском сервера
python test_runner.py --test all --auto-start

# С ожиданием готовности
python test_runner.py --test all --wait-timeout 120
```

## Отчеты

### Типы отчетов

1. **Индивидуальные отчеты** - `reports/{test_name}_YYYYMMDD_HHMMSS.md`
2. **Сводные отчеты** - `reports/chaos_test_summary_YYYYMMDD_HHMMSS.md`
3. **Аналитические отчеты** - `reports/analysis_YYYYMMDD_HHMMSS.md`
4. **HTML дашборд** - `reports/dashboard.html`
5. **Графики** - `reports/success_rate_chart.png`, `reports/test_statistics_chart.png`

### Просмотр отчетов

```bash
# Открыть дашборд
start reports/dashboard.html  # Windows
open reports/dashboard.html    # macOS
xdg-open reports/dashboard.html # Linux

# Просмотр последнего отчета
cat reports/chaos_test_summary_*.md | tail -1
```

## Диагностика

### Проверка сервера

```powershell
# PowerShell
.\diagnose_server.ps1

# Python
python test_connection.py
```

### Ручной запуск сервера

```powershell
cd E:\HttpServer
$env:ARLIAI_API_KEY="597dbe7e-16ca-4803-ab17-5fa084909f37"
.\httpserver_no_gui.exe
```

## Параметры командной строки

### Общие параметры

- `--base-url URL` - Базовый URL сервера (по умолчанию: http://localhost:9999)
- `--quick` - Быстрый режим (меньше запросов)
- `--auto-start` - Автоматически запустить сервер
- `--wait-timeout SECONDS` - Таймаут ожидания сервера

### Специфичные параметры

- `--test NAME` - Выбор теста для запуска
- `--reports-dir DIR` - Директория для отчетов
- `--output-dir DIR` - Директория для вывода

## Интерпретация результатов

### Успешные тесты (✅ PASSED)

- Все проверки пройдены
- Нет критических ошибок
- Система работает стабильно

### Проваленные тесты (❌ FAILED)

- Обнаружены проблемы
- Требуется анализ детального отчета
- Возможны race conditions или ошибки обработки

### Предупреждения (⚠️ WARNING)

- Потенциальные проблемы
- Не критичные, но требуют внимания
- Рекомендуется мониторинг

## Рекомендации

### Регулярное тестирование

```bash
# Ежедневный запуск
python run_complete_suite.py --auto-start --quick

# Полный запуск раз в неделю
python run_complete_suite.py --auto-start
```

### Анализ трендов

```bash
# Анализ истории
python report_analyzer.py

# Просмотр дашборда
start reports/dashboard.html
```

### Устранение проблем

1. Проверьте детальные отчеты
2. Изучите логи в `logs/`
3. Запустите диагностику сервера
4. Проверьте переменные окружения

## Дополнительная информация

- **README.md** - Основная документация
- **QUICKSTART.md** - Быстрый старт
- **USAGE.md** - Руководство по использованию
- **DEVELOPMENT_PROGRESS.md** - Прогресс разработки
- **BUILD_ANALYSIS.md** - Анализ сборки

## Поддержка

При возникновении проблем:

1. Проверьте логи в `logs/`
2. Запустите диагностику: `.\diagnose_server.ps1`
3. Проверьте документацию в `TROUBLESHOOTING.md`
4. Изучите отчеты в `reports/`

