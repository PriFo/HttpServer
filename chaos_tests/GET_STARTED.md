# Chaos Monkey Testing - Начните здесь

## 🎯 Быстрый старт

### Шаг 1: Установите Python (если не установлен)

**Windows:**
```powershell
# Через winget
winget install Python.Python.3.11

# Или скачайте с python.org
# https://www.python.org/downloads/
```

**Проверка:**
```powershell
python --version
# или
py --version
```

### Шаг 2: Установите зависимости

```powershell
cd E:\HttpServer\chaos_tests
pip install requests psutil
```

### Шаг 3: Запустите сервер

```powershell
cd E:\HttpServer
.\httpserver_no_gui.exe
```

Дождитесь сообщения: `✓ Сервер успешно запущен на порту 9999`

### Шаг 4: Запустите тесты

**Способ 1: Умный скрипт (рекомендуется)**
```powershell
cd E:\HttpServer
.\chaos_tests\run_tests_windows.ps1 all
```

**Способ 2: Автозапуск сервера и тестов**
```powershell
cd E:\HttpServer
.\chaos_tests\start_server_and_test.ps1 all
```

**Способ 3: Вручную**
```powershell
cd E:\HttpServer\chaos_tests
python chaos_monkey.py --test all
```

## 📋 Чек-лист перед запуском

- [ ] Python установлен и доступен
- [ ] Зависимости установлены (`pip install requests psutil`)
- [ ] Сервер запущен на localhost:9999
- [ ] Сервер отвечает на запросы (HTTP 200, не 502)

## 🔍 Проверка готовности

### Проверка Python
```powershell
python --version
python -c "import requests; import psutil; print('OK')"
```

### Проверка сервера
```powershell
Invoke-WebRequest http://localhost:9999/api/config
```

### Быстрая проверка
```powershell
python chaos_tests/test_connection.py
```

## ⚠️ Решение проблем

Если что-то не работает, смотрите:
- `TROUBLESHOOTING.md` - детальные решения проблем
- `RUN_INSTRUCTIONS.md` - инструкции по запуску
- `EXECUTION_REPORT.md` - отчет о выполнении

## 📊 Что дальше

После успешного запуска:
1. Проверьте отчеты в `chaos_tests/reports/`
2. Изучите логи в `chaos_tests/logs/`
3. Исправьте найденные проблемы
4. Повторите тесты

---

**Готово к использованию!** 🚀

