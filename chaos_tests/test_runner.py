#!/usr/bin/env python3
"""
Улучшенный запуск тестов Chaos Monkey
С автоматической проверкой сервера и улучшенной обработкой ошибок
"""

import sys
import time
import subprocess
from pathlib import Path

try:
    import requests
except ImportError:
    print("Error: 'requests' library is not installed.")
    print("Install it with: pip install requests")
    sys.exit(1)


def check_server(base_url: str = "http://localhost:9999", max_attempts: int = 10) -> bool:
    """Проверка доступности сервера"""
    print(f"Проверка доступности сервера {base_url}...")
    
    for attempt in range(1, max_attempts + 1):
        try:
            response = requests.get(f"{base_url}/health", timeout=3)
            if response.status_code == 200:
                print(f"✅ Сервер доступен!")
                return True
        except requests.exceptions.RequestException:
            if attempt < max_attempts:
                print(f"   Попытка {attempt}/{max_attempts}... ожидание...")
                time.sleep(3)
            else:
                print(f"❌ Сервер недоступен после {max_attempts} попыток")
                return False
    
    return False


def wait_for_server(base_url: str = "http://localhost:9999", timeout: int = 60) -> bool:
    """Ожидание запуска сервера с таймаутом"""
    print(f"Ожидание запуска сервера (таймаут: {timeout} секунд)...")
    start_time = time.time()
    
    while time.time() - start_time < timeout:
        if check_server(base_url, max_attempts=1):
            return True
        time.sleep(2)
    
    print(f"❌ Сервер не запустился за {timeout} секунд")
    return False


def start_server_if_needed() -> bool:
    """Попытка запустить сервер, если он не запущен"""
    if check_server(max_attempts=1):
        return True
    
    print("Сервер не запущен. Попытка запуска...")
    
    # Ищем исполняемый файл сервера
    script_dir = Path(__file__).parent
    root_dir = script_dir.parent
    server_exe = root_dir / "httpserver_no_gui.exe"
    
    if not server_exe.exists():
        print(f"❌ Файл сервера не найден: {server_exe}")
        return False
    
    try:
        # Запускаем сервер в фоне
        import os
        env = os.environ.copy()
        env['ARLIAI_API_KEY'] = env.get('ARLIAI_API_KEY', '597dbe7e-16ca-4803-ab17-5fa084909f37')
        env['CGO_ENABLED'] = '1'
        
        subprocess.Popen(
            [str(server_exe)],
            cwd=str(root_dir),
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL
        )
        
        print("Сервер запущен. Ожидание готовности...")
        return wait_for_server(timeout=30)
        
    except Exception as e:
        print(f"❌ Ошибка при запуске сервера: {e}")
        return False


def main():
    """Главная функция"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Chaos Monkey Test Runner')
    parser.add_argument('--test', type=str, default='all',
                       choices=['all', 'concurrent_config', 'invalid_normalization', 
                               'ai_failure', 'large_data'],
                       help='Тест для запуска')
    parser.add_argument('--base-url', type=str, default='http://localhost:9999',
                       help='Базовый URL сервера')
    parser.add_argument('--quick', action='store_true',
                       help='Быстрый режим (меньше запросов)')
    parser.add_argument('--auto-start', action='store_true',
                       help='Автоматически запустить сервер, если он не запущен')
    parser.add_argument('--wait-timeout', type=int, default=60,
                       help='Таймаут ожидания сервера (секунды)')
    
    args = parser.parse_args()
    
    # Проверка/запуск сервера
    if args.auto_start:
        if not start_server_if_needed():
            print("\n❌ Не удалось запустить сервер автоматически")
            print("Запустите сервер вручную:")
            print("  cd E:\\HttpServer")
            print("  $env:ARLIAI_API_KEY='597dbe7e-16ca-4803-ab17-5fa084909f37'")
            print("  .\\httpserver_no_gui.exe")
            sys.exit(1)
    else:
        if not check_server(args.base_url, max_attempts=1):
            print("\n⚠️ Сервер недоступен!")
            print("Варианты:")
            print("  1. Запустите сервер вручную")
            print("  2. Используйте --auto-start для автоматического запуска")
            sys.exit(1)
    
    # Запуск тестов
    print(f"\n{'=' * 60}")
    print(f"Запуск тестов Chaos Monkey")
    print(f"{'=' * 60}\n")
    
    # Импортируем и запускаем chaos_monkey
    script_dir = Path(__file__).parent
    chaos_monkey_path = script_dir / "chaos_monkey.py"
    
    cmd = [sys.executable, str(chaos_monkey_path)]
    cmd.extend(['--test', args.test])
    cmd.extend(['--base-url', args.base_url])
    if args.quick:
        cmd.append('--quick')
    
    try:
        result = subprocess.run(cmd, check=False)
        sys.exit(result.returncode)
    except KeyboardInterrupt:
        print("\n\nПрервано пользователем")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Ошибка при запуске тестов: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()

