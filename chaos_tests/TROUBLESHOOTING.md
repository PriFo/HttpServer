# Решение проблем - Chaos Monkey Testing

## Проблема 1: Python не найден

### Симптомы
```
python: The term 'python' is not recognized
```

### Решения

#### Вариант A: Установка Python
```powershell
# Через winget
winget install Python.Python.3.11

# Или скачайте с python.org
# https://www.python.org/downloads/
```

#### Вариант B: Использование найденного Python
```powershell
# Используйте полный путь
C:\Users\eugin\.local\bin\python3.11.exe chaos_tests/chaos_monkey.py --test all
```

#### Вариант C: Использование скрипта-обертки
```powershell
# Используйте готовый скрипт
.\chaos_tests\run_tests_windows.ps1 all
```

## Проблема 2: Сервер возвращает 502 Bad Gateway

### Симптомы
```
Response status code does not indicate success: 502 (Bad Gateway)
```

### Причины и решения

#### 1. Проблемы с конфигурацией
**Решение:**
- Проверьте файл конфигурации
- Убедитесь, что все пути к БД корректны
- Проверьте переменные окружения

#### 2. Недоступность базы данных
**Решение:**
```powershell
# Проверьте наличие БД
Test-Path ".\1c_data.db"
Test-Path ".\service.db"

# Проверьте права доступа
Get-Acl ".\1c_data.db"
```

#### 3. Ошибки инициализации
**Решение:**
```powershell
# Проверьте логи сервера
Get-Content backend.log -Tail 50
Get-Content backend.err -Tail 50
```

#### 4. Порт занят другим процессом
**Решение:**
```powershell
# Проверьте, что порт свободен
netstat -ano | findstr ":9999"

# Остановите конфликтующий процесс
Stop-Process -Id <PID> -Force
```

## Проблема 3: WSL не может подключиться к localhost

### Симптомы
```
Connection refused
Failed to establish a new connection: [Errno 111]
```

### Решения

#### Вариант A: Использование IP адреса Windows хоста
```bash
# В WSL найдите IP Windows хоста
ip route show | grep default | awk '{print $3}'

# Используйте его в тестах
export CHAOS_BASE_URL="http://<WINDOWS_IP>:9999"
python3 chaos_monkey.py --test all
```

#### Вариант B: Использование Python напрямую в Windows
```powershell
# Используйте скрипт-обертку
.\chaos_tests\run_tests_windows.ps1 all
```

#### Вариант C: Настройка WSL networking
```bash
# В WSL добавьте в /etc/hosts
echo "$(ip route | awk '/^default/ {print $3}') host.docker.internal" | sudo tee -a /etc/hosts
```

## Проблема 4: Python ошибка "No module named 'encodings'"

### Симптомы
```
Fatal Python error: init_fs_encoding: failed to get the Python codec
ModuleNotFoundError: No module named 'encodings'
```

### Решения

#### Вариант A: Переустановка Python
```powershell
# Удалите проблемный Python
# Установите заново с python.org
```

#### Вариант B: Использование другого Python
```powershell
# Используйте системный Python
py -3 chaos_tests/chaos_monkey.py --test all

# Или через WSL
wsl python3 chaos_tests/chaos_monkey.py --test all
```

#### Вариант C: Использование виртуального окружения
```powershell
# Создайте venv
python -m venv venv
.\venv\Scripts\Activate.ps1
pip install requests psutil
python chaos_tests/chaos_monkey.py --test all
```

## Проблема 5: Зависимости не установлены

### Симптомы
```
ModuleNotFoundError: No module named 'requests'
ModuleNotFoundError: No module named 'psutil'
```

### Решение
```powershell
# Установите зависимости
pip install requests psutil

# Или через requirements.txt
pip install -r chaos_tests/requirements.txt
```

## Проблема 6: Множественные процессы сервера

### Симптомы
```
Несколько процессов httpserver_no_gui.exe запущены
```

### Решение
```powershell
# Остановите все процессы
Get-Process -Name "httpserver*" | Stop-Process -Force

# Или используйте скрипт
.\chaos_tests\start_server_and_test.ps1
```

## Рекомендуемый порядок действий

1. **Используйте готовые скрипты:**
   ```powershell
   .\chaos_tests\start_server_and_test.ps1 all
   ```

2. **Если не работает, проверьте вручную:**
   ```powershell
   # Проверка Python
   python --version
   
   # Проверка сервера
   Invoke-WebRequest http://localhost:9999/health
   
   # Проверка зависимостей
   python -c "import requests; import psutil; print('OK')"
   ```

3. **Используйте альтернативные методы:**
   - WSL с правильным IP
   - Виртуальное окружение Python
   - Docker (если доступен)

## Быстрая диагностика

Запустите диагностический скрипт:
```bash
bash chaos_tests/diagnostic.sh
```

Или проверку подключения:
```powershell
python chaos_tests/test_connection.py
```

## Получение помощи

1. Проверьте логи: `chaos_tests/logs/`
2. Проверьте отчеты: `chaos_tests/reports/`
3. Проверьте документацию: `chaos_tests/README.md`
4. Проверьте инструкции: `chaos_tests/RUN_INSTRUCTIONS.md`

