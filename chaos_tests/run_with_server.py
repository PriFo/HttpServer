#!/usr/bin/env python3
"""
Скрипт для автоматического запуска сервера и тестов Chaos Monkey
"""

import os
import sys
import time
import subprocess
import signal
import requests
from pathlib import Path

BASE_URL = "http://localhost:9999"
SERVER_EXECUTABLES = [
    "../httpserver_no_gui.exe",
    "./httpserver_no_gui.exe",
    "../httpserver.exe",
    "./httpserver.exe",
    "../bin/httpserver_no_gui.exe",
]

def find_server_executable():
    """Поиск исполняемого файла сервера"""
    for exe in SERVER_EXECUTABLES:
        if os.path.exists(exe):
            return os.path.abspath(exe)
    return None

def check_server_running():
    """Проверка, запущен ли сервер"""
    try:
        response = requests.get(f"{BASE_URL}/api/config", timeout=2)
        return response.status_code == 200
    except:
        return False

def start_server(server_exe):
    """Запуск сервера"""
    print(f"🚀 Запуск сервера: {server_exe}")
    
    # Запускаем сервер в фоне
    process = subprocess.Popen(
        [server_exe],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        cwd=os.path.dirname(server_exe) or "."
    )
    
    # Ждем запуска сервера
    print("⏳ Ожидание запуска сервера...")
    for i in range(30):  # Максимум 30 секунд
        time.sleep(1)
        if check_server_running():
            print(f"✅ Сервер запущен и доступен на {BASE_URL}")
            return process
        if process.poll() is not None:
            # Процесс завершился
            stdout, stderr = process.communicate()
            print(f"❌ Сервер завершился с ошибкой:")
            print(f"STDOUT: {stdout.decode('utf-8', errors='ignore')}")
            print(f"STDERR: {stderr.decode('utf-8', errors='ignore')}")
            return None
    
    print("⚠️ Сервер не ответил в течение 30 секунд, но процесс запущен")
    return process

def stop_server(process):
    """Остановка сервера"""
    if process and process.poll() is None:
        print("\n🛑 Остановка сервера...")
        try:
            if sys.platform == "win32":
                process.terminate()
            else:
                process.send_signal(signal.SIGTERM)
            
            # Ждем завершения
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
            
            print("✅ Сервер остановлен")
        except Exception as e:
            print(f"⚠️ Ошибка при остановке сервера: {e}")

def main():
    """Главная функция"""
    print("=" * 60)
    print("Chaos Monkey Testing - Автоматический запуск")
    print("=" * 60)
    print()
    
    # Проверяем, запущен ли сервер
    if check_server_running():
        print(f"✅ Сервер уже запущен на {BASE_URL}")
        server_process = None
        need_stop = False
    else:
        # Ищем исполняемый файл сервера
        server_exe = find_server_executable()
        if not server_exe:
            print("❌ Не найден исполняемый файл сервера")
            print("Искали в:")
            for exe in SERVER_EXECUTABLES:
                print(f"  - {exe}")
            print("\nУбедитесь, что сервер скомпилирован и находится в одном из этих мест.")
            return 1
        
        # Запускаем сервер
        server_process = start_server(server_exe)
        if not server_process:
            return 1
        need_stop = True
    
    try:
        # Запускаем тесты
        print("\n" + "=" * 60)
        print("Запуск Chaos Monkey тестов")
        print("=" * 60)
        print()
        
        # Определяем, какой тест запускать
        test_name = "all"
        if len(sys.argv) > 1:
            test_name = sys.argv[1]
        
        # Запускаем chaos_monkey.py
        script_dir = Path(__file__).parent
        chaos_script = script_dir / "chaos_monkey.py"
        
        if not chaos_script.exists():
            print(f"❌ Не найден скрипт: {chaos_script}")
            return 1
        
        result = subprocess.run(
            [sys.executable, str(chaos_script), "--test", test_name],
            cwd=str(script_dir)
        )
        
        return result.returncode
        
    finally:
        # Останавливаем сервер, если мы его запускали
        if need_stop and server_process:
            stop_server(server_process)

if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("\n\n⚠️ Прервано пользователем")
        sys.exit(1)

