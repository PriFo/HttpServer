#!/usr/bin/env python3
"""
Быстрый тест для проверки работоспособности системы
"""

import sys
from pathlib import Path

# Добавляем текущую директорию в путь
sys.path.insert(0, str(Path(__file__).parent))

try:
    from test_connection import test_connection
    from chaos_monkey import ConcurrentConfigTest, setup_logging, BaseTest
except ImportError as e:
    print(f"Ошибка импорта: {e}")
    print("Убедитесь, что все зависимости установлены: pip install requests")
    sys.exit(1)


def main():
    """Быстрый тест"""
    print("=" * 60)
    print("Chaos Monkey - Быстрый тест")
    print("=" * 60)
    print()
    
    # Шаг 1: Проверка подключения
    print("Шаг 1: Проверка подключения к серверу")
    print("-" * 60)
    if not test_connection():
        print("\n❌ Сервер недоступен. Запустите сервер и повторите попытку.")
        return 1
    print()
    
    # Шаг 2: Быстрый тест конкурентных обновлений
    print("Шаг 2: Быстрый тест конкурентных обновлений")
    print("-" * 60)
    
    log_dir = Path("./logs")
    report_dir = Path("./reports")
    logger = setup_logging(log_dir)
    
    test = ConcurrentConfigTest("http://localhost:9999", logger, report_dir)
    success = test.run()
    
    print()
    print("=" * 60)
    if success:
        print("✅ Быстрый тест пройден успешно")
    else:
        print("❌ Быстрый тест провален")
        print("Проверьте отчеты в chaos_tests/reports/")
    print("=" * 60)
    
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())

