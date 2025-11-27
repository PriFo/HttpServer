#!/usr/bin/env python3
"""
Стресс-тесты для проверки устойчивости системы под нагрузкой
"""

import time
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Dict, List, Tuple
import requests
import logging


class StressTest:
    """Класс для стресс-тестирования"""
    
    def __init__(self, base_url: str, logger: logging.Logger):
        self.base_url = base_url.rstrip('/')
        self.logger = logger
        self.session = requests.Session()
        self.session.timeout = 30
        self.results = {
            'total_requests': 0,
            'successful': 0,
            'failed': 0,
            'errors': [],
            'response_times': [],
            'http_codes': {}
        }
    
    def make_request(self, endpoint: str, method: str = 'GET', data: Dict = None) -> Tuple[int, float]:
        """Выполнение одного запроса"""
        url = f"{self.base_url}{endpoint}"
        start_time = time.time()
        
        try:
            if method == 'GET':
                response = self.session.get(url)
            elif method == 'POST':
                response = self.session.post(url, json=data)
            else:
                response = self.session.get(url)
            
            elapsed = time.time() - start_time
            status_code = response.status_code
            
            self.results['total_requests'] += 1
            self.results['response_times'].append(elapsed)
            
            if status_code not in self.results['http_codes']:
                self.results['http_codes'][status_code] = 0
            self.results['http_codes'][status_code] += 1
            
            if 200 <= status_code < 300:
                self.results['successful'] += 1
            else:
                self.results['failed'] += 1
                self.results['errors'].append(f"{endpoint}: HTTP {status_code}")
            
            return status_code, elapsed
            
        except Exception as e:
            elapsed = time.time() - start_time
            self.results['total_requests'] += 1
            self.results['failed'] += 1
            self.results['errors'].append(f"{endpoint}: {str(e)}")
            return 0, elapsed
    
    def stress_test_endpoint(self, endpoint: str, concurrent: int = 20, 
                            requests_per_thread: int = 10, method: str = 'GET') -> Dict:
        """Стресс-тест конкретного endpoint"""
        self.logger.info(f"Стресс-тест: {method} {endpoint}")
        self.logger.info(f"Параллельных потоков: {concurrent}, запросов на поток: {requests_per_thread}")
        
        start_time = time.time()
        
        def worker():
            results = []
            for _ in range(requests_per_thread):
                status, elapsed = self.make_request(endpoint, method)
                results.append((status, elapsed))
            return results
        
        with ThreadPoolExecutor(max_workers=concurrent) as executor:
            futures = [executor.submit(worker) for _ in range(concurrent)]
            for future in as_completed(futures):
                future.result()
        
        total_time = time.time() - start_time
        total_requests = concurrent * requests_per_thread
        
        # Статистика
        if self.results['response_times']:
            avg_time = sum(self.results['response_times']) / len(self.results['response_times'])
            min_time = min(self.results['response_times'])
            max_time = max(self.results['response_times'])
            rps = total_requests / total_time if total_time > 0 else 0
        else:
            avg_time = min_time = max_time = rps = 0
        
        stats = {
            'endpoint': endpoint,
            'method': method,
            'total_requests': total_requests,
            'successful': self.results['successful'],
            'failed': self.results['failed'],
            'success_rate': (self.results['successful'] / total_requests * 100) if total_requests > 0 else 0,
            'total_time': total_time,
            'avg_response_time': avg_time,
            'min_response_time': min_time,
            'max_response_time': max_time,
            'requests_per_second': rps,
            'http_codes': self.results['http_codes'].copy()
        }
        
        self.logger.info(f"Результаты: {stats['successful']}/{total_requests} успешных, "
                        f"RPS: {rps:.2f}, среднее время: {avg_time:.3f}s")
        
        return stats


def run_stress_tests(base_url: str = "http://localhost:9999"):
    """Запуск стресс-тестов"""
    logging.basicConfig(level=logging.INFO, format='%(levelname)s: %(message)s')
    logger = logging.getLogger('StressTest')
    
    test = StressTest(base_url, logger)
    
    logger.info("=" * 60)
    logger.info("Стресс-тестирование API")
    logger.info("=" * 60)
    
    endpoints = [
        ('/api/config', 'GET'),
        ('/api/clients', 'GET'),
        ('/api/databases/list', 'GET'),
    ]
    
    results = []
    for endpoint, method in endpoints:
        try:
            stats = test.stress_test_endpoint(endpoint, concurrent=10, 
                                            requests_per_thread=5, method=method)
            results.append(stats)
            time.sleep(2)  # Пауза между тестами
        except Exception as e:
            logger.error(f"Ошибка при тестировании {endpoint}: {e}")
    
    # Итоговый отчет
    logger.info("\n" + "=" * 60)
    logger.info("Итоги стресс-тестирования")
    logger.info("=" * 60)
    
    for stats in results:
        logger.info(f"\n{stats['method']} {stats['endpoint']}:")
        logger.info(f"  Успешных: {stats['successful']}/{stats['total_requests']} "
                   f"({stats['success_rate']:.1f}%)")
        logger.info(f"  RPS: {stats['requests_per_second']:.2f}")
        logger.info(f"  Среднее время: {stats['avg_response_time']:.3f}s")
        logger.info(f"  HTTP коды: {stats['http_codes']}")
    
    return results


if __name__ == "__main__":
    import sys
    base_url = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:9999"
    run_stress_tests(base_url)

