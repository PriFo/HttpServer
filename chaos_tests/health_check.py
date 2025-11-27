#!/usr/bin/env python3
"""
Health check для Chaos Monkey тестов
Проверка состояния системы перед запуском тестов
"""

import sys
import time
from pathlib import Path
from typing import Dict, List, Tuple

try:
    import requests
except ImportError:
    print("Error: 'requests' library is not installed.")
    sys.exit(1)

try:
    import psutil
    PSUTIL_AVAILABLE = True
except ImportError:
    PSUTIL_AVAILABLE = False


class HealthChecker:
    """Проверка состояния системы"""
    
    def __init__(self, base_url: str = "http://localhost:9999"):
        self.base_url = base_url.rstrip('/')
        self.checks = []
        self.results = {}
    
    def check_server_health(self) -> Tuple[bool, str]:
        """Проверка здоровья сервера"""
        try:
            response = requests.get(f"{self.base_url}/health", timeout=5)
            if response.status_code == 200:
                return True, "Сервер отвечает на /health"
            else:
                return False, f"Сервер вернул статус {response.status_code}"
        except requests.exceptions.ConnectionError:
            return False, "Не удалось подключиться к серверу"
        except requests.exceptions.Timeout:
            return False, "Таймаут при подключении к серверу"
        except Exception as e:
            return False, f"Ошибка: {str(e)}"
    
    def check_api_config(self) -> Tuple[bool, str]:
        """Проверка доступности API конфигурации"""
        try:
            response = requests.get(f"{self.base_url}/api/config", timeout=5)
            if response.status_code == 200:
                data = response.json()
                if isinstance(data, dict):
                    return True, "API конфигурации доступно"
                else:
                    return False, "API вернуло невалидный JSON"
            else:
                return False, f"API вернуло статус {response.status_code}"
        except Exception as e:
            return False, f"Ошибка: {str(e)}"
    
    def check_server_process(self) -> Tuple[bool, str]:
        """Проверка процесса сервера"""
        if not PSUTIL_AVAILABLE:
            return None, "psutil не установлен"
        
        try:
            for proc in psutil.process_iter(['pid', 'name', 'memory_info']):
                if 'httpserver' in proc.info['name'].lower():
                    mem_mb = proc.info['memory_info'].rss / 1024 / 1024
                    return True, f"Процесс найден (PID: {proc.pid}, Memory: {mem_mb:.1f}MB)"
            return False, "Процесс сервера не найден"
        except Exception as e:
            return False, f"Ошибка: {str(e)}"
    
    def check_disk_space(self, path: Path = Path('.')) -> Tuple[bool, str]:
        """Проверка свободного места на диске"""
        if not PSUTIL_AVAILABLE:
            return None, "psutil не установлен"
        
        try:
            usage = psutil.disk_usage(str(path))
            free_gb = usage.free / (1024 ** 3)
            total_gb = usage.total / (1024 ** 3)
            percent_free = (usage.free / usage.total) * 100
            
            if percent_free < 10:
                return False, f"Мало места на диске: {free_gb:.1f}GB свободно ({percent_free:.1f}%)"
            else:
                return True, f"Свободно места: {free_gb:.1f}GB из {total_gb:.1f}GB ({percent_free:.1f}%)"
        except Exception as e:
            return False, f"Ошибка: {str(e)}"
    
    def check_dependencies(self) -> Tuple[bool, str]:
        """Проверка зависимостей"""
        missing = []
        
        try:
            import requests
        except ImportError:
            missing.append('requests')
        
        try:
            import psutil
        except ImportError:
            missing.append('psutil (опционально)')
        
        if missing:
            return False, f"Отсутствуют зависимости: {', '.join(missing)}"
        else:
            return True, "Все зависимости установлены"
    
    def run_all_checks(self) -> Dict:
        """Запуск всех проверок"""
        checks = [
            ('Сервер /health', self.check_server_health),
            ('API /api/config', self.check_api_config),
            ('Процесс сервера', self.check_server_process),
            ('Свободное место', self.check_disk_space),
            ('Зависимости', self.check_dependencies),
        ]
        
        results = {}
        all_passed = True
        
        print("Выполнение проверок здоровья системы...\n")
        
        for name, check_func in checks:
            passed, message = check_func()
            results[name] = {
                'passed': passed,
                'message': message
            }
            
            if passed is False:
                all_passed = False
            
            status = "✅" if passed else ("❌" if passed is False else "⚠️")
            print(f"{status} {name}: {message}")
        
        print()
        
        if all_passed:
            print("✅ Все проверки пройдены!")
        else:
            print("❌ Некоторые проверки не пройдены")
        
        return {
            'all_passed': all_passed,
            'checks': results
        }


def main():
    """Главная функция"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Health check для Chaos Monkey тестов')
    parser.add_argument('--base-url', type=str, default='http://localhost:9999',
                       help='Базовый URL сервера')
    parser.add_argument('--json', action='store_true',
                       help='Вывод в формате JSON')
    
    args = parser.parse_args()
    
    checker = HealthChecker(args.base_url)
    results = checker.run_all_checks()
    
    if args.json:
        import json
        print(json.dumps(results, indent=2, ensure_ascii=False))
    
    sys.exit(0 if results['all_passed'] else 1)


if __name__ == '__main__':
    main()

