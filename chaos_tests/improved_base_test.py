#!/usr/bin/env python3
"""
Улучшенный базовый класс для тестов с дополнительной диагностикой
"""

import json
import logging
import time
from pathlib import Path
from typing import Dict, List, Optional, Tuple
import requests


class ImprovedBaseTest:
    """Улучшенный базовый класс с расширенной диагностикой"""
    
    def __init__(self, base_url: str, logger: logging.Logger, report_dir: Path):
        self.base_url = base_url.rstrip('/')
        self.logger = logger
        self.report_dir = report_dir
        self.report_dir.mkdir(parents=True, exist_ok=True)
        self.session = requests.Session()
        self.session.timeout = 30
        self.results = {
            'passed': 0,
            'failed': 0,
            'warnings': 0,
            'errors': [],
            'timings': [],
            'http_codes': []
        }
        self.test_start_time = time.time()
    
    def check_server_health(self) -> bool:
        """Проверка здоровья сервера перед тестами"""
        self.logger.info("Проверка доступности сервера...")
        
        try:
            response = self.session.get(f"{self.base_url}/health", timeout=5)
            if response.status_code == 200:
                self.logger.info("✅ Сервер доступен")
                return True
            else:
                self.logger.warning(f"⚠️ Health check вернул {response.status_code}")
                return False
        except requests.exceptions.ConnectionError:
            self.logger.error("❌ Сервер недоступен - connection refused")
            self.logger.error(f"Убедитесь, что сервер запущен на {self.base_url}")
            return False
        except Exception as e:
            self.logger.error(f"❌ Ошибка при проверке сервера: {e}")
            return False
    
    def http_request(self, method: str, endpoint: str, data: Optional[Dict] = None,
                    headers: Optional[Dict] = None, retries: int = 3,
                    expected_status: Optional[int] = None) -> Tuple[int, Dict, float]:
        """Выполнение HTTP запроса с улучшенной обработкой ошибок"""
        url = f"{self.base_url}{endpoint}"
        request_headers = headers or {}
        request_headers.setdefault('Content-Type', 'application/json')
        
        start_time = time.time()
        last_error = None
        
        for attempt in range(retries):
            try:
                if method.upper() == 'GET':
                    response = self.session.get(url, headers=request_headers, timeout=30)
                elif method.upper() == 'POST':
                    response = self.session.post(url, json=data, headers=request_headers, timeout=30)
                elif method.upper() == 'PUT':
                    response = self.session.put(url, json=data, headers=request_headers, timeout=30)
                else:
                    raise ValueError(f"Unsupported method: {method}")
                
                elapsed = time.time() - start_time
                self.results['timings'].append(elapsed)
                self.results['http_codes'].append(response.status_code)
                
                try:
                    response_data = response.json()
                except ValueError:
                    response_data = {'text': response.text[:500]}
                
                # Проверка ожидаемого статуса
                if expected_status and response.status_code != expected_status:
                    self.logger.warning(
                        f"Неожиданный статус для {endpoint}: "
                        f"ожидался {expected_status}, получен {response.status_code}"
                    )
                
                # Логирование нестандартных статусов
                if response.status_code >= 500:
                    self.logger.warning(
                        f"Server error {response.status_code} on {endpoint} "
                        f"(attempt {attempt + 1}/{retries})"
                    )
                elif response.status_code == 502:
                    self.logger.error(
                        f"Bad Gateway (502) - возможно проблема с прокси или конфигурацией сервера"
                    )
                
                return response.status_code, response_data, elapsed
                
            except requests.exceptions.ConnectionError as e:
                last_error = e
                if attempt < retries - 1:
                    wait_time = (attempt + 1) * 2
                    self.logger.debug(
                        f"Connection error, retrying in {wait_time}s... "
                        f"({attempt + 1}/{retries})"
                    )
                    time.sleep(wait_time)
                else:
                    elapsed = time.time() - start_time
                    error_msg = f'Connection failed after {retries} attempts: {str(e)}'
                    self.logger.error(error_msg)
                    self.results['errors'].append(error_msg)
                    return 0, {'error': error_msg}, elapsed
                    
            except requests.exceptions.Timeout as e:
                last_error = e
                if attempt < retries - 1:
                    self.logger.warning(f"Timeout, retrying... ({attempt + 1}/{retries})")
                    time.sleep(2)
                else:
                    elapsed = time.time() - start_time
                    error_msg = f'Timeout after {retries} attempts: {str(e)}'
                    self.logger.error(error_msg)
                    self.results['errors'].append(error_msg)
                    return 0, {'error': error_msg}, elapsed
                    
            except requests.exceptions.RequestException as e:
                elapsed = time.time() - start_time
                error_msg = f'Request failed: {str(e)}'
                self.logger.error(error_msg)
                self.results['errors'].append(error_msg)
                return 0, {'error': error_msg}, elapsed
        
        elapsed = time.time() - start_time
        error_msg = f'Failed after {retries} attempts: {last_error}'
        self.results['errors'].append(error_msg)
        return 0, {'error': error_msg}, elapsed
    
    def check_response(self, status_code: int, expected: int = 200,
                      error_msg: Optional[str] = None) -> bool:
        """Проверка статуса ответа"""
        if status_code == expected:
            self.results['passed'] += 1
            return True
        else:
            self.results['failed'] += 1
            msg = error_msg or f"Expected {expected}, got {status_code}"
            self.results['errors'].append(msg)
            self.logger.warning(msg)
            return False
    
    def get_statistics(self) -> Dict:
        """Получение статистики по тесту"""
        total_time = time.time() - self.test_start_time
        timings = self.results.get('timings', [])
        
        stats = {
            'total_time': total_time,
            'total_requests': len(timings),
            'passed': self.results['passed'],
            'failed': self.results['failed'],
            'warnings': self.results['warnings'],
        }
        
        if timings:
            stats['avg_response_time'] = sum(timings) / len(timings)
            stats['min_response_time'] = min(timings)
            stats['max_response_time'] = max(timings)
        
        http_codes = self.results.get('http_codes', [])
        if http_codes:
            from collections import Counter
            stats['http_code_distribution'] = dict(Counter(http_codes))
        
        return stats
    
    def generate_report(self, test_name: str, description: str) -> Path:
        """Генерация улучшенного отчета"""
        report_file = self.report_dir / f"{test_name}_{time.strftime('%Y%m%d_%H%M%S')}.md"
        stats = self.get_statistics()
        
        with open(report_file, 'w', encoding='utf-8') as f:
            f.write(f"# Отчет: {test_name}\n\n")
            f.write(f"**Дата:** {time.strftime('%Y-%m-%d %H:%M:%S')}\n")
            f.write(f"**Сервер:** {self.base_url}\n\n")
            f.write(f"## Описание\n\n{description}\n\n")
            
            f.write("## Статистика\n\n")
            f.write(f"- **Общее время:** {stats['total_time']:.2f}s\n")
            f.write(f"- **Всего запросов:** {stats['total_requests']}\n")
            if 'avg_response_time' in stats:
                f.write(f"- **Среднее время ответа:** {stats['avg_response_time']:.3f}s\n")
                f.write(f"- **Мин. время ответа:** {stats['min_response_time']:.3f}s\n")
                f.write(f"- **Макс. время ответа:** {stats['max_response_time']:.3f}s\n")
            f.write(f"- **Пройдено:** {stats['passed']}\n")
            f.write(f"- **Провалено:** {stats['failed']}\n")
            f.write(f"- **Предупреждения:** {stats['warnings']}\n\n")
            
            if 'http_code_distribution' in stats:
                f.write("## Распределение HTTP кодов\n\n")
                for code, count in sorted(stats['http_code_distribution'].items()):
                    f.write(f"- HTTP {code}: {count} раз\n")
                f.write("\n")
            
            if self.results['errors']:
                f.write("## Ошибки\n\n")
                for error in self.results['errors']:
                    f.write(f"- {error}\n")
                f.write("\n")
        
        return report_file

