#!/usr/bin/env python3
"""
Простой скрипт для проверки подключения к серверу
"""

import sys
import requests
import time

BASE_URL = "http://localhost:9999"

def test_connection():
    """Проверка подключения к серверу"""
    print("=" * 60)
    print("Проверка подключения к серверу")
    print("=" * 60)
    print(f"URL: {BASE_URL}")
    print()
    
    # Тест 1: Health check
    print("1. Проверка health endpoint...")
    try:
        response = requests.get(f"{BASE_URL}/health", timeout=5)
        print(f"   ✅ Health check: HTTP {response.status_code}")
        if response.status_code == 200:
            print(f"   Ответ: {response.text[:100]}")
    except requests.exceptions.ConnectionError:
        print("   ❌ Connection refused - сервер не запущен или недоступен")
        return False
    except Exception as e:
        print(f"   ⚠️ Ошибка: {e}")
    
    print()
    
    # Тест 2: Config endpoint
    print("2. Проверка config endpoint...")
    try:
        response = requests.get(f"{BASE_URL}/api/config", timeout=5)
        print(f"   Статус: HTTP {response.status_code}")
        
        if response.status_code == 200:
            print("   ✅ Config endpoint доступен")
            try:
                config = response.json()
                print(f"   Порт: {config.get('port', 'N/A')}")
                print(f"   Database: {config.get('database_path', 'N/A')}")
            except:
                print("   ⚠️ Ответ не является валидным JSON")
        elif response.status_code == 502:
            print("   ❌ Bad Gateway - сервер запущен, но есть проблемы с прокси/конфигурацией")
        elif response.status_code >= 500:
            print(f"   ❌ Server Error {response.status_code}")
        else:
            print(f"   ⚠️ Неожиданный статус: {response.status_code}")
            
    except requests.exceptions.ConnectionError:
        print("   ❌ Connection refused")
        return False
    except Exception as e:
        print(f"   ❌ Ошибка: {e}")
        return False
    
    print()
    
    # Тест 3: Проверка времени ответа
    print("3. Проверка времени ответа...")
    times = []
    for i in range(5):
        try:
            start = time.time()
            response = requests.get(f"{BASE_URL}/api/config", timeout=5)
            elapsed = time.time() - start
            times.append(elapsed)
            if response.status_code == 200:
                print(f"   Запрос {i+1}: {elapsed:.3f}s ✅")
            else:
                print(f"   Запрос {i+1}: {elapsed:.3f}s ⚠️ (HTTP {response.status_code})")
        except Exception as e:
            print(f"   Запрос {i+1}: ❌ {e}")
    
    if times:
        avg_time = sum(times) / len(times)
        print(f"   Среднее время ответа: {avg_time:.3f}s")
        if avg_time > 1.0:
            print("   ⚠️ Медленный ответ - возможны проблемы с производительностью")
    
    print()
    print("=" * 60)
    print("Проверка завершена")
    print("=" * 60)
    
    return True

if __name__ == "__main__":
    success = test_connection()
    sys.exit(0 if success else 1)

