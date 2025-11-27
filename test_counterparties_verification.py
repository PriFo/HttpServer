#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Скрипт для проверки функциональности контрагентов
Проверяет API и фронтенд
"""

import requests
import json
import sys
from urllib.parse import urlencode, quote

BACKEND_URL = "http://localhost:9999"
FRONTEND_URL = "http://localhost:3000"
TIMEOUT = 7

def test_server(url, name):
    """Проверка доступности сервера"""
    try:
        response = requests.get(f"{url}/health", timeout=TIMEOUT)
        if response.status_code == 200:
            print(f"[OK] {name} доступен")
            return True
        else:
            print(f"[ERROR] {name} вернул код {response.status_code}")
            return False
    except Exception as e:
        print(f"[ERROR] {name} недоступен: {e}")
        return False

def main():
    print("=" * 50)
    print("Проверка функциональности контрагентов")
    print("=" * 50)
    print()
    
    # 1. Проверка доступности серверов
    print("1. Проверка доступности серверов...")
    backend_available = test_server(BACKEND_URL, "Backend")
    frontend_available = test_server(FRONTEND_URL, "Frontend")
    
    if not backend_available:
        print()
        print("ВНИМАНИЕ: Backend недоступен. Убедитесь, что сервер запущен на порту 9999")
        print("Для запуска используйте: go run main_no_gui.go")
        print()
        sys.exit(1)
    
    if not frontend_available:
        print()
        print("ВНИМАНИЕ: Frontend недоступен. Убедитесь, что сервер запущен на порту 3000")
        print("Для запуска используйте: cd frontend && npm run dev")
        print()
    
    print()
    print("2. Проверка API endpoints...")
    print()
    
    # 2.1. Получение списка клиентов
    print("2.1. Получение списка клиентов...")
    try:
        response = requests.get(f"{BACKEND_URL}/api/clients", timeout=TIMEOUT)
        response.raise_for_status()
        clients = response.json()
        
        if clients and len(clients) > 0:
            first_client = clients[0]
            client_id = first_client['id']
            print(f"  [OK] Найден клиент с ID: {client_id}")
            print(f"  Имя: {first_client.get('name', 'N/A')}")
        else:
            print("  [WARNING] Клиенты не найдены")
            print("  Используем тестовый client_id=1")
            client_id = 1
    except Exception as e:
        print(f"  [ERROR] Не удалось получить список клиентов: {e}")
        print("  Используем тестовый client_id=1")
        client_id = 1
    
    # 2.2. Проверка основного endpoint
    print()
    print(f"2.2. Проверка /api/counterparties/normalized?client_id={client_id}...")
    try:
        params = {"client_id": client_id}
        url = f"{BACKEND_URL}/api/counterparties/normalized?{urlencode(params)}"
        response = requests.get(url, timeout=TIMEOUT)
        response.raise_for_status()
        data = response.json()
        
        print("  [OK] Запрос выполнен успешно")
        print("  Структура ответа:")
        print(f"    - counterparties: {len(data.get('counterparties', []))} записей")
        print(f"    - total: {data.get('total', 0)}")
        print(f"    - projects: {len(data.get('projects', []))} проектов")
        print(f"    - offset: {data.get('offset', 0)}")
        print(f"    - limit: {data.get('limit', 0)}")
        print(f"    - page: {data.get('page', 0)}")
        
        counterparties = data.get('counterparties', [])
        if len(counterparties) > 0:
            print("  [OK] Контрагенты найдены")
            first_counterparty = counterparties[0]
            print("  Пример контрагента:")
            print(f"    - ID: {first_counterparty.get('id', 'N/A')}")
            print(f"    - Название: {first_counterparty.get('name', 'N/A')}")
            if first_counterparty.get('normalized_name'):
                print(f"    - Нормализованное название: {first_counterparty.get('normalized_name')}")
        else:
            print(f"  [WARNING] Контрагенты не найдены для клиента {client_id}")
    except Exception as e:
        print(f"  [ERROR] Ошибка запроса: {e}")
        if hasattr(e, 'response') and e.response is not None:
            try:
                error_data = e.response.json()
                print(f"  Ответ сервера: {json.dumps(error_data, indent=2, ensure_ascii=False)}")
            except:
                print(f"  Ответ сервера: {e.response.text}")
    
    # 2.3. Проверка пагинации
    print()
    print("2.3. Проверка пагинации (page=1&limit=10)...")
    try:
        params = {"client_id": client_id, "page": 1, "limit": 10}
        url = f"{BACKEND_URL}/api/counterparties/normalized?{urlencode(params)}"
        response = requests.get(url, timeout=TIMEOUT)
        response.raise_for_status()
        page_data = response.json()
        
        print("  [OK] Пагинация работает")
        print(f"    - Получено записей: {len(page_data.get('counterparties', []))}")
        print(f"    - Лимит: {page_data.get('limit', 0)}")
        print(f"    - Страница: {page_data.get('page', 0)}")
        print(f"    - Всего: {page_data.get('total', 0)}")
        
        counterparties_count = len(page_data.get('counterparties', []))
        limit = page_data.get('limit', 0)
        if counterparties_count <= limit:
            print("  [OK] Лимит соблюдается")
        else:
            print("  [WARNING] Получено больше записей, чем указано в лимите")
    except Exception as e:
        print(f"  [ERROR] Ошибка проверки пагинации: {e}")
    
    # 2.4. Проверка поиска
    print()
    print("2.4. Проверка поиска (search=тест)...")
    try:
        params = {"client_id": client_id, "search": "тест"}
        url = f"{BACKEND_URL}/api/counterparties/normalized?{urlencode(params)}"
        response = requests.get(url, timeout=TIMEOUT)
        response.raise_for_status()
        search_data = response.json()
        
        print("  [OK] Поиск работает")
        print(f"    - Найдено записей: {len(search_data.get('counterparties', []))}")
        print(f"    - Всего: {search_data.get('total', 0)}")
    except Exception as e:
        print(f"  [ERROR] Ошибка проверки поиска: {e}")
    
    # 2.5. Проверка фильтра по проекту
    print()
    print("2.5. Проверка фильтра по проекту...")
    try:
        params = {"client_id": client_id}
        url = f"{BACKEND_URL}/api/counterparties/normalized?{urlencode(params)}"
        response = requests.get(url, timeout=TIMEOUT)
        response.raise_for_status()
        projects_data = response.json()
        
        projects = projects_data.get('projects', [])
        if projects and len(projects) > 0:
            project_id = projects[0]['id']
            try:
                params = {"client_id": client_id, "project_id": project_id}
                url = f"{BACKEND_URL}/api/counterparties/normalized?{urlencode(params)}"
                response = requests.get(url, timeout=TIMEOUT)
                response.raise_for_status()
                filtered_data = response.json()
                
                print("  [OK] Фильтр по проекту работает")
                print(f"    - Проект ID: {project_id}")
                print(f"    - Найдено записей: {len(filtered_data.get('counterparties', []))}")
                print(f"    - Всего: {filtered_data.get('total', 0)}")
            except Exception as e:
                print(f"  [ERROR] Ошибка проверки фильтра: {e}")
        else:
            print("  [SKIP] Нет проектов для проверки фильтра")
    except Exception as e:
        print(f"  [ERROR] Не удалось получить список проектов: {e}")
    
    # 3. Проверка обработки ошибок
    print()
    print("3. Проверка обработки ошибок...")
    print()
    
    # 3.1. Несуществующий клиент
    print("3.1. Проверка несуществующего клиента...")
    try:
        params = {"client_id": 999999}
        url = f"{BACKEND_URL}/api/counterparties/normalized?{urlencode(params)}"
        response = requests.get(url, timeout=TIMEOUT)
        response.raise_for_status()
        data = response.json()
        
        if len(data.get('counterparties', [])) == 0:
            print("  [OK] Корректно обработан несуществующий клиент (пустой список)")
        else:
            print("  [WARNING] Для несуществующего клиента возвращены данные")
    except requests.exceptions.HTTPError as e:
        if e.response.status_code in [404, 400]:
            print("  [OK] Корректная обработка ошибки для несуществующего клиента")
        else:
            print(f"  [ERROR] Неожиданная ошибка: {e}")
    except Exception as e:
        print(f"  [ERROR] Неожиданная ошибка: {e}")
    
    # 3.2. Запрос без client_id
    print()
    print("3.2. Проверка запроса без client_id...")
    try:
        url = f"{BACKEND_URL}/api/counterparties/normalized"
        response = requests.get(url, timeout=TIMEOUT)
        response.raise_for_status()
        print("  [WARNING] Запрос без client_id не вернул ошибку")
    except requests.exceptions.HTTPError as e:
        if e.response.status_code == 400:
            print("  [OK] Корректная обработка ошибки (400 Bad Request)")
        else:
            print(f"  [WARNING] Неожиданный код ошибки: {e.response.status_code}")
    except Exception as e:
        print(f"  [ERROR] Ошибка: {e}")
    
    print()
    print("=" * 50)
    print("Проверка завершена")
    print("=" * 50)
    print()
    
    if frontend_available:
        print("Для проверки фронтенда откройте:")
        print(f"  {FRONTEND_URL}/clients/{client_id}")
        print("  и перейдите на вкладку Контрагенты")
        print()

if __name__ == "__main__":
    main()

