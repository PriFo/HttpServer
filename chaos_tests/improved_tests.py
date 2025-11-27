#!/usr/bin/env python3
"""
Улучшенные тесты Chaos Monkey
Дополнительные тесты и улучшенная обработка ошибок
"""

import json
import logging
import time
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from concurrent.futures import ThreadPoolExecutor, as_completed

import sys

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


class DatabaseLockTest:
    """Тест на блокировки базы данных при конкурентных операциях"""
    
    def __init__(self, base_url: str, logger: logging.Logger):
        self.base_url = base_url.rstrip('/')
        self.logger = logger
        self.session = requests.Session()
        self.session.timeout = 30
    
    def run_concurrent_updates(self, num_threads: int = 20) -> Dict:
        """Запуск конкурентных обновлений конфигурации"""
        self.logger.info(f"Запуск {num_threads} конкурентных обновлений конфигурации...")
        
        results = {
            'total': num_threads,
            'success': 0,
            'failed': 0,
            'database_locked': 0,
            'other_errors': 0,
            'errors': []
        }
        
        def update_config(index: int) -> Tuple[int, Optional[str]]:
            """Обновление конфигурации с индексом"""
            try:
                config = {
                    'port': f'999{index % 10}',
                    'test_index': index,
                    'timestamp': time.time()
                }
                
                response = self.session.put(
                    f"{self.base_url}/api/config",
                    json=config,
                    timeout=10
                )
                
                if response.status_code == 200:
                    return (index, None)
                elif 'database is locked' in response.text.lower() or response.status_code == 500:
                    return (index, 'database_locked')
                else:
                    return (index, f"HTTP {response.status_code}: {response.text[:100]}")
                    
            except Exception as e:
                return (index, str(e))
        
        # Запуск конкурентных обновлений
        with ThreadPoolExecutor(max_workers=num_threads) as executor:
            futures = [executor.submit(update_config, i) for i in range(num_threads)]
            
            for future in as_completed(futures):
                index, error = future.result()
                if error is None:
                    results['success'] += 1
                elif error == 'database_locked':
                    results['database_locked'] += 1
                    results['failed'] += 1
                else:
                    results['other_errors'] += 1
                    results['failed'] += 1
                    results['errors'].append(f"Thread {index}: {error}")
        
        return results
    
    def check_config_history(self) -> Dict:
        """Проверка истории конфигурации на пропуски версий"""
        self.logger.info("Проверка истории конфигурации...")
        
        try:
            # Получаем текущую конфигурацию
            response = self.session.get(f"{self.base_url}/api/config")
            if response.status_code != 200:
                return {'error': f"Failed to get config: {response.status_code}"}
            
            current_config = response.json()
            current_version = current_config.get('version', 0)
            
            # Проверяем историю (если есть endpoint)
            # Это зависит от реализации API
            return {
                'current_version': current_version,
                'note': 'History check requires API endpoint'
            }
            
        except Exception as e:
            return {'error': str(e)}


class StressTest:
    """Стресс-тест API endpoints"""
    
    def __init__(self, base_url: str, logger: logging.Logger):
        self.base_url = base_url.rstrip('/')
        self.logger = logger
        self.session = requests.Session()
        self.session.timeout = 30
    
    def stress_endpoint(self, endpoint: str, method: str = 'GET', 
                       data: Optional[Dict] = None, 
                       num_requests: int = 100,
                       concurrency: int = 10) -> Dict:
        """Стресс-тест конкретного endpoint"""
        self.logger.info(f"Стресс-тест {method} {endpoint} ({num_requests} запросов, {concurrency} потоков)")
        
        results = {
            'total': num_requests,
            'success': 0,
            'failed': 0,
            'timeouts': 0,
            'errors': [],
            'response_times': [],
            'status_codes': {}
        }
        
        def make_request(index: int) -> Tuple[int, Optional[Dict]]:
            """Выполнение одного запроса"""
            start_time = time.time()
            try:
                if method == 'GET':
                    response = self.session.get(
                        f"{self.base_url}{endpoint}",
                        timeout=5
                    )
                elif method == 'POST':
                    response = self.session.post(
                        f"{self.base_url}{endpoint}",
                        json=data or {},
                        timeout=5
                    )
                else:
                    return (index, {'error': f'Unsupported method: {method}'})
                
                elapsed = time.time() - start_time
                status = response.status_code
                
                results['response_times'].append(elapsed)
                results['status_codes'][status] = results['status_codes'].get(status, 0) + 1
                
                if 200 <= status < 300:
                    return (index, {'status': status, 'time': elapsed})
                else:
                    return (index, {'status': status, 'error': response.text[:100], 'time': elapsed})
                    
            except requests.exceptions.Timeout:
                results['timeouts'] += 1
                return (index, {'error': 'timeout'})
            except Exception as e:
                return (index, {'error': str(e)})
        
        # Запуск стресс-теста
        start_time = time.time()
        with ThreadPoolExecutor(max_workers=concurrency) as executor:
            futures = [executor.submit(make_request, i) for i in range(num_requests)]
            
            for future in as_completed(futures):
                index, result = future.result()
                if result and 'error' not in result:
                    results['success'] += 1
                else:
                    results['failed'] += 1
                    if result:
                        results['errors'].append(f"Request {index}: {result.get('error', 'unknown')}")
        
        total_time = time.time() - start_time
        
        # Статистика
        if results['response_times']:
            results['avg_response_time'] = sum(results['response_times']) / len(results['response_times'])
            results['min_response_time'] = min(results['response_times'])
            results['max_response_time'] = max(results['response_times'])
        
        results['total_time'] = total_time
        results['requests_per_second'] = num_requests / total_time if total_time > 0 else 0
        
        return results


class ResourceMonitor:
    """Мониторинг ресурсов во время тестов"""
    
    def __init__(self, logger: logging.Logger):
        self.logger = logger
        self.monitoring = False
        self.data = []
    
    def start_monitoring(self, process_name: str = "httpserver_no_gui"):
        """Начало мониторинга"""
        if not PSUTIL_AVAILABLE:
            self.logger.warning("psutil не установлен, мониторинг ресурсов недоступен")
            return
        
        self.monitoring = True
        self.data = []
        
        # Находим процесс
        processes = []
        for proc in psutil.process_iter(['pid', 'name']):
            try:
                if process_name.lower() in proc.info['name'].lower():
                    processes.append(proc)
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                pass
        
        if not processes:
            self.logger.warning(f"Процесс {process_name} не найден")
            return
        
        self.process = processes[0]
        self.logger.info(f"Мониторинг процесса: {self.process.info['name']} (PID: {self.process.pid})")
    
    def collect_sample(self):
        """Сбор одного образца данных"""
        if not self.monitoring or not PSUTIL_AVAILABLE:
            return
        
        try:
            proc = self.process
            with proc.oneshot():
                cpu_percent = proc.cpu_percent()
                memory_info = proc.memory_info()
                memory_percent = proc.memory_percent()
                
                self.data.append({
                    'timestamp': time.time(),
                    'cpu_percent': cpu_percent,
                    'memory_mb': memory_info.rss / 1024 / 1024,
                    'memory_percent': memory_percent
                })
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            self.monitoring = False
    
    def stop_monitoring(self) -> Dict:
        """Остановка мониторинга и возврат статистики"""
        self.monitoring = False
        
        if not self.data:
            return {'error': 'No data collected'}
        
        cpu_values = [d['cpu_percent'] for d in self.data if d['cpu_percent'] is not None]
        memory_values = [d['memory_mb'] for d in self.data]
        
        return {
            'samples': len(self.data),
            'cpu': {
                'avg': sum(cpu_values) / len(cpu_values) if cpu_values else 0,
                'max': max(cpu_values) if cpu_values else 0,
                'min': min(cpu_values) if cpu_values else 0
            },
            'memory': {
                'avg_mb': sum(memory_values) / len(memory_values) if memory_values else 0,
                'max_mb': max(memory_values) if memory_values else 0,
                'min_mb': min(memory_values) if memory_values else 0
            },
            'data': self.data
        }


def main():
    """Пример использования улучшенных тестов"""
    import argparse
    
    logging.basicConfig(level=logging.INFO, format='%(levelname)s: %(message)s')
    logger = logging.getLogger(__name__)
    
    parser = argparse.ArgumentParser()
    parser.add_argument('--base-url', default='http://localhost:9999')
    parser.add_argument('--test', choices=['db_lock', 'stress', 'monitor'], required=True)
    
    args = parser.parse_args()
    
    if args.test == 'db_lock':
        test = DatabaseLockTest(args.base_url, logger)
        results = test.run_concurrent_updates(num_threads=20)
        logger.info(f"Результаты: {json.dumps(results, indent=2)}")
        
    elif args.test == 'stress':
        test = StressTest(args.base_url, logger)
        results = test.stress_endpoint('/api/config', method='GET', num_requests=100)
        logger.info(f"Результаты: {json.dumps(results, indent=2)}")
        
    elif args.test == 'monitor':
        monitor = ResourceMonitor(logger)
        monitor.start_monitoring()
        
        # Мониторинг в течение 30 секунд
        for _ in range(30):
            monitor.collect_sample()
            time.sleep(1)
        
        stats = monitor.stop_monitoring()
        logger.info(f"Статистика: {json.dumps(stats, indent=2)}")


if __name__ == '__main__':
    main()

