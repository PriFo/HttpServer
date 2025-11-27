#!/usr/bin/env python3
"""
Запуск всех интегрированных тестов Chaos Monkey
"""

import sys
from pathlib import Path

# Добавляем текущую директорию в путь
script_dir = Path(__file__).parent
sys.path.insert(0, str(script_dir))

from test_runner import check_server, wait_for_server
from integrated_chaos_monkey import main as integrated_main


def main():
    """Главная функция"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Run all integrated Chaos Monkey tests')
    parser.add_argument('--base-url', type=str, default='http://localhost:9999',
                       help='Base URL of the server')
    parser.add_argument('--auto-start', action='store_true',
                       help='Automatically start server if not running')
    parser.add_argument('--wait-timeout', type=int, default=60,
                       help='Timeout for waiting server (seconds)')
    parser.add_argument('--quick', action='store_true',
                       help='Quick mode (fewer requests)')
    
    args = parser.parse_args()
    
    # Проверка/ожидание сервера
    print("Проверка доступности сервера...")
    if not check_server(args.base_url, max_attempts=1):
        if args.auto_start:
            from test_runner import start_server_if_needed
            if not start_server_if_needed():
                print("❌ Не удалось запустить сервер")
                sys.exit(1)
        else:
            print("⚠️ Сервер недоступен!")
            print("Используйте --auto-start для автоматического запуска")
            if not wait_for_server(args.base_url, timeout=args.wait_timeout):
                sys.exit(1)
    
    # Запуск всех тестов
    sys.argv = ['integrated_chaos_monkey.py', '--test', 'all', '--base-url', args.base_url]
    if args.quick:
        sys.argv.append('--quick')
    
    integrated_main()


if __name__ == '__main__':
    main()

