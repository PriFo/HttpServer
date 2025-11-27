#!/usr/bin/env python3
"""
Запуск всех расширенных тестов включая стресс-тесты и проверку целостности
"""

import sys
import subprocess
from pathlib import Path


def run_test(script_name: str, description: str) -> bool:
    """Запуск теста"""
    print(f"\n{'=' * 60}")
    print(f"{description}")
    print(f"{'=' * 60}\n")
    
    script_path = Path(__file__).parent / script_name
    
    if not script_path.exists():
        print(f"❌ Скрипт не найден: {script_name}")
        return False
    
    try:
        result = subprocess.run(
            [sys.executable, str(script_path)],
            capture_output=True,
            text=True,
            timeout=300  # 5 минут максимум
        )
        
        print(result.stdout)
        if result.stderr:
            print("STDERR:", result.stderr)
        
        return result.returncode == 0
        
    except subprocess.TimeoutExpired:
        print(f"⏱️ Тест превысил лимит времени (5 минут)")
        return False
    except Exception as e:
        print(f"❌ Ошибка при запуске теста: {e}")
        return False


def main():
    """Главная функция"""
    print("=" * 60)
    print("Chaos Monkey - Расширенное тестирование")
    print("=" * 60)
    
    base_url = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:9999"
    print(f"Сервер: {base_url}\n")
    
    tests = [
        ('test_connection.py', 'Проверка подключения'),
        ('chaos_monkey.py', 'Основные Chaos Monkey тесты', ['--test', 'all']),
        ('test_stress.py', 'Стресс-тестирование'),
        ('test_endpoint_coverage.py', 'Проверка покрытия endpoints'),
        ('test_data_integrity.py', 'Проверка целостности данных'),
    ]
    
    results = {}
    
    for test_info in tests:
        script_name = test_info[0]
        description = test_info[1]
        extra_args = test_info[2] if len(test_info) > 2 else []
        
        success = run_test(script_name, description)
        results[script_name] = success
        
        if not success:
            print(f"\n⚠️ Тест {script_name} завершился с ошибками")
    
    # Итоги
    print("\n" + "=" * 60)
    print("Итоги расширенного тестирования")
    print("=" * 60)
    
    passed = sum(1 for v in results.values() if v)
    total = len(results)
    
    for script, success in results.items():
        status = "✅" if success else "❌"
        print(f"{status} {script}")
    
    print(f"\nВсего: {passed}/{total} тестов пройдено")
    
    return 0 if passed == total else 1


if __name__ == "__main__":
    sys.exit(main())

