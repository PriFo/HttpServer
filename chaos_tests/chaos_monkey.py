#!/usr/bin/env python3
"""
Chaos Monkey Backend Testing - Python Version
Кросс-платформенный инструмент для нагрузочного и нестандартного тестирования бэкенда
"""

import argparse
import json
import logging
import os
import sys
import time
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple, Any

try:
    import requests
except ImportError:
    print("Error: 'requests' library is not installed.")
    print("Install it with: pip install requests")
    sys.exit(1)

try:
    import psutil
    PSUTIL_AVAILABLE = True
except ImportError:
    PSUTIL_AVAILABLE = False
    print("Warning: 'psutil' library is not installed. Resource monitoring will be limited.")
    print("Install it with: pip install psutil")


# Настройка логирования
class ColoredFormatter(logging.Formatter):
    """Форматтер с цветным выводом для консоли"""
    
    COLORS = {
        'DEBUG': '\033[0;36m',    # Cyan
        'INFO': '\033[0;34m',     # Blue
        'WARNING': '\033[1;33m',   # Yellow
        'ERROR': '\033[0;31m',    # Red
        'CRITICAL': '\033[1;31m', # Bold Red
    }
    RESET = '\033[0m'
    
    def format(self, record):
        log_color = self.COLORS.get(record.levelname, '')
        record.levelname = f"{log_color}{record.levelname}{self.RESET}"
        return super().format(record)


def setup_logging(log_dir: Path) -> logging.Logger:
    """Настройка логирования с выводом в консоль и файл"""
    log_dir.mkdir(parents=True, exist_ok=True)
    log_file = log_dir / f"chaos_monkey_{datetime.now().strftime('%Y%m%d')}.log"
    
    logger = logging.getLogger('ChaosMonkey')
    logger.setLevel(logging.DEBUG)
    
    # Консольный handler с цветами
    console_handler = logging.StreamHandler()
    console_handler.setLevel(logging.INFO)
    console_formatter = ColoredFormatter('%(levelname)s: %(message)s')
    console_handler.setFormatter(console_formatter)
    
    # Файловый handler
    file_handler = logging.FileHandler(log_file, encoding='utf-8')
    file_handler.setLevel(logging.DEBUG)
    file_formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    file_handler.setFormatter(file_formatter)
    
    logger.addHandler(console_handler)
    logger.addHandler(file_handler)
    
    return logger


class BaseTest:
    """Базовый класс для всех тестов"""
    
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
            'errors': []
        }
    
    def http_request(self, method: str, endpoint: str, data: Optional[Dict] = None,
                    headers: Optional[Dict] = None, retries: int = 3) -> Tuple[int, Dict, float]:
        """Выполнение HTTP запроса с логированием и retry логикой"""
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
                
                try:
                    response_data = response.json()
                except ValueError:
                    response_data = {'text': response.text[:500]}  # Ограничиваем размер
                
                # Логируем нестандартные статусы
                if response.status_code >= 500:
                    self.logger.warning(f"Server error {response.status_code} on {endpoint} (attempt {attempt + 1}/{retries})")
                elif response.status_code >= 400:
                    self.logger.debug(f"Client error {response.status_code} on {endpoint}")
                
                return response.status_code, response_data, elapsed
                
            except requests.exceptions.ConnectionError as e:
                last_error = e
                if attempt < retries - 1:
                    wait_time = (attempt + 1) * 2  # Экспоненциальная задержка
                    self.logger.debug(f"Connection error, retrying in {wait_time}s... ({attempt + 1}/{retries})")
                    time.sleep(wait_time)
                else:
                    elapsed = time.time() - start_time
                    self.logger.error(f"Connection failed after {retries} attempts: {e}")
                    return 0, {'error': f'Connection failed: {str(e)}'}, elapsed
                    
            except requests.exceptions.Timeout as e:
                last_error = e
                if attempt < retries - 1:
                    self.logger.warning(f"Timeout, retrying... ({attempt + 1}/{retries})")
                    time.sleep(2)
                else:
                    elapsed = time.time() - start_time
                    self.logger.error(f"Request timeout after {retries} attempts: {e}")
                    return 0, {'error': f'Timeout: {str(e)}'}, elapsed
                    
            except requests.exceptions.RequestException as e:
                elapsed = time.time() - start_time
                self.logger.error(f"Request failed: {e}")
                return 0, {'error': str(e)}, elapsed
        
        elapsed = time.time() - start_time
        return 0, {'error': f'Failed after {retries} attempts: {last_error}'}, elapsed
    
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
    
    def get_config(self) -> Optional[Dict]:
        """Получение текущей конфигурации"""
        status, data, _ = self.http_request('GET', '/api/config')
        if status == 200:
            return data
        return None
    
    def save_config(self, config: Dict) -> bool:
        """Сохранение конфигурации"""
        status, _, _ = self.http_request('PUT', '/api/config', data=config)
        return status == 200
    
    def generate_report(self, test_name: str, description: str) -> Path:
        """Генерация отчета по тесту"""
        report_file = self.report_dir / f"{test_name}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
        
        with open(report_file, 'w', encoding='utf-8') as f:
            f.write(f"# Отчет: {test_name}\n\n")
            f.write(f"**Дата:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
            f.write(f"**Сервер:** {self.base_url}\n\n")
            f.write(f"## Описание\n\n{description}\n\n")
            f.write(f"## Результаты\n\n")
            f.write(f"- **Пройдено:** {self.results['passed']}\n")
            f.write(f"- **Провалено:** {self.results['failed']}\n")
            f.write(f"- **Предупреждения:** {self.results['warnings']}\n\n")
            
            if self.results['errors']:
                f.write("## Ошибки\n\n")
                for error in self.results['errors']:
                    f.write(f"- {error}\n")
                f.write("\n")
        
        return report_file


class ConcurrentConfigTest(BaseTest):
    """Тест конкурентных обновлений конфигурации"""
    
    def run(self) -> bool:
        """Запуск теста"""
        self.logger.info("=== Тест: Конкурентные обновления конфигурации ===")
        
        # Получаем исходную конфигурацию
        original_config = self.get_config()
        if not original_config:
            self.logger.error("Не удалось получить исходную конфигурацию")
            self.logger.error("Проверьте, что сервер запущен и доступен на " + self.base_url)
            self.results['errors'].append("Не удалось получить исходную конфигурацию - сервер недоступен")
            return False
        
        self.logger.info("Исходная конфигурация получена")
        
        # Функция для одного запроса обновления
        def update_config(index: int) -> Tuple[int, Dict, float]:
            test_config = original_config.copy()
            test_config['port'] = f"999{index}"
            return self.http_request('PUT', '/api/config', data=test_config)
        
        # Определяем количество параллельных запросов
        num_requests = 5 if hasattr(self, 'quick_mode') and self.quick_mode else 10
        self.logger.info(f"Запуск {num_requests} параллельных обновлений конфигурации...")
        db_locked_count = 0
        
        with ThreadPoolExecutor(max_workers=num_requests) as executor:
            futures = [executor.submit(update_config, i) for i in range(num_requests)]
            results = []
            
            for future in as_completed(futures):
                status, data, elapsed = future.result()
                results.append((status, data, elapsed))
                
                if status == 200:
                    self.check_response(status, 200)
                else:
                    error_text = json.dumps(data).lower()
                    if 'locked' in error_text or 'database' in error_text:
                        db_locked_count += 1
                    self.check_response(status, 200, f"HTTP {status}: {data}")
        
        # Проверяем историю конфигурации
        self.logger.info("Проверка истории конфигурации...")
        status, history_data, _ = self.http_request('GET', '/api/config/history?limit=20')
        
        if status == 200 and 'history' in history_data:
            versions = [h.get('version') for h in history_data['history'] if h.get('version')]
            versions.sort()
            
            gaps = 0
            for i in range(1, len(versions)):
                if versions[i] - versions[i-1] > 1:
                    gaps += 1
                    self.logger.warning(f"Пропуск версий: {versions[i-1]} -> {versions[i]}")
            
            if gaps == 0:
                self.logger.info("Пропусков версий не обнаружено")
        
        # Восстанавливаем исходную конфигурацию
        self.logger.info("Восстановление исходной конфигурации...")
        self.save_config(original_config)
        
        # Генерируем отчет
        description = """
1. Сохранена исходная конфигурация
2. Запущено 10 параллельных PUT /api/config запросов
3. Проанализированы результаты на наличие ошибок блокировки БД
4. Проверена история конфигурации на пропуски версий
5. Восстановлена исходная конфигурация
        """
        
        report_file = self.generate_report("concurrent_config", description)
        self.logger.info(f"Отчет сохранен: {report_file}")
        
        if db_locked_count > 0:
            self.logger.warning(f"Обнаружено {db_locked_count} ошибок блокировки БД")
        
        return self.results['failed'] == 0


class InvalidNormalizationTest(BaseTest):
    """Тест нормализации с невалидными данными"""
    
    def run(self) -> bool:
        """Запуск теста"""
        self.logger.info("=== Тест: Нормализация с невалидными данными ===")
        
        # Получаем клиента и проект
        status, clients_data, _ = self.http_request('GET', '/api/clients')
        if status != 200 or not clients_data:
            self.logger.warning("Не удалось получить список клиентов, используем значения по умолчанию")
            client_id, project_id = 1, 1
        else:
            clients = clients_data if isinstance(clients_data, list) else clients_data.get('clients', [])
            if clients:
                client_id = clients[0].get('id', 1)
                # Получаем проекты клиента
                status, projects_data, _ = self.http_request('GET', f'/api/clients/{client_id}/projects')
                if status == 200:
                    projects = projects_data if isinstance(projects_data, list) else projects_data.get('projects', [])
                    project_id = projects[0].get('id', 1) if projects else 1
                else:
                    project_id = 1
            else:
                client_id, project_id = 1, 1
        
        self.logger.info(f"Используется client_id={client_id}, project_id={project_id}")
        
        # Тест 1: Несуществующий database_id
        self.logger.info("Тест 1: Несуществующий database_id")
        status, data, _ = self.http_request('POST',
            f'/api/clients/{client_id}/projects/{project_id}/normalization/start',
            data={'database_ids': [999999], 'all_active': False}
        )
        self.check_response(status, 400, f"Ожидался 4xx, получен {status}")
        
        # Тест 2: Невалидный item_id (0)
        self.logger.info("Тест 2: Невалидный item_id (0)")
        status, data, _ = self.http_request('POST', '/api/normalization/start',
            data={'item_id': 0, 'original_name': 'Test Item'}
        )
        self.check_response(status, 400, f"Ожидался 400, получен {status}")
        
        # Тест 3: Пустой original_name
        self.logger.info("Тест 3: Пустой original_name")
        status, data, _ = self.http_request('POST', '/api/normalization/start',
            data={'item_id': 123, 'original_name': ''}
        )
        self.check_response(status, 400, f"Ожидался 400, получен {status}")
        
        # Тест 4: Отрицательный лимит
        self.logger.info("Тест 4: Отрицательный лимит в истории")
        status, data, _ = self.http_request('GET', '/api/normalization/history?limit=-10')
        # Может быть либо ошибка, либо игнорирование невалидного параметра
        if status >= 400:
            self.check_response(status, 400)
        else:
            self.check_response(status, 200)
        
        # Тест 5: Несуществующий session_id
        self.logger.info("Тест 5: Несуществующий session_id")
        status, data, _ = self.http_request('GET', '/api/normalization/session/999999')
        self.check_response(status, 404, f"Ожидался 404, получен {status}")
        
        # Генерируем отчет
        description = """
1. Тест с несуществующим database_id
2. Тест с невалидным item_id (0)
3. Тест с пустым original_name
4. Тест с отрицательным лимитом в истории
5. Тест с несуществующим session_id
        """
        
        report_file = self.generate_report("invalid_normalization", description)
        self.logger.info(f"Отчет сохранен: {report_file}")
        
        return self.results['failed'] == 0


class AIFailureTest(BaseTest):
    """Тест устойчивости к сбоям AI сервиса"""
    
    def run(self) -> bool:
        """Запуск теста"""
        self.logger.info("=== Тест: Устойчивость к сбоям AI сервиса ===")
        
        # Сохраняем исходную конфигурацию
        original_config = self.get_config()
        if not original_config:
            self.logger.error("Не удалось получить исходную конфигурацию")
            return False
        
        original_api_key = original_config.get('arliai_api_key', '')
        
        # Устанавливаем невалидный API ключ
        self.logger.info("Установка невалидного API ключа...")
        invalid_config = original_config.copy()
        invalid_config['arliai_api_key'] = 'invalid_key_12345'
        self.save_config(invalid_config)
        time.sleep(2)
        
        # Создаем сессию нормализации
        self.logger.info("Создание сессии нормализации...")
        status, data, _ = self.http_request('POST', '/api/normalization/start',
            data={'item_id': 1, 'original_name': 'Тестовый товар для проверки AI'}
        )
        
        if status == 200 and 'session_id' in data:
            session_id = data['session_id']
            self.logger.info(f"Сессия создана: {session_id}")
            
            # Применяем паттерны
            self.logger.info("Применение паттернов...")
            time.sleep(1)
            status, data, _ = self.http_request('POST', '/api/normalization/apply-patterns',
                data={'session_id': session_id}
            )
            self.check_response(status, 200, f"Применение паттернов: HTTP {status}")
            
            # Применяем AI коррекцию (должна вернуть ошибку)
            self.logger.info("Применение AI коррекции с невалидным ключом...")
            time.sleep(1)
            status, data, _ = self.http_request('POST', '/api/normalization/apply-ai',
                data={'session_id': session_id, 'use_chat': False}
            )
            
            if status >= 400:
                self.check_response(status, 400, f"AI коррекция вернула ошибку: HTTP {status}")
            else:
                self.logger.warning(f"Неожиданный статус для AI коррекции: {status}")
                self.results['warnings'] += 1
            
            # Проверяем статус сессии
            self.logger.info("Проверка статуса сессии...")
            time.sleep(1)
            status, session_data, _ = self.http_request('GET', f'/api/normalization/session/{session_id}')
            
            if status == 200:
                session_status = session_data.get('status', 'unknown')
                if session_status in ['failed', 'error']:
                    self.logger.info(f"Сессия имеет статус '{session_status}' (ожидалось)")
                    self.check_response(200, 200)
                else:
                    self.logger.warning(f"Сессия имеет статус '{session_status}' (ожидался 'failed' или 'error')")
                    self.results['warnings'] += 1
        else:
            self.logger.warning(f"Не удалось создать сессию: HTTP {status}")
            self.results['warnings'] += 1
        
        # Восстанавливаем исходную конфигурацию
        self.logger.info("Восстановление исходной конфигурации...")
        if original_api_key:
            restored_config = original_config.copy()
            restored_config['arliai_api_key'] = original_api_key
        else:
            restored_config = original_config.copy()
            restored_config.pop('arliai_api_key', None)
        
        self.save_config(restored_config)
        time.sleep(2)
        
        # Генерируем отчет
        description = """
1. Сохранена исходная конфигурация
2. Установлен невалидный ARLIAI_API_KEY
3. Запущен полный цикл нормализации (start -> apply-patterns -> apply-ai)
4. Проверен статус сессии после ошибки AI
5. Восстановлен валидный API ключ
        """
        
        report_file = self.generate_report("ai_failure", description)
        self.logger.info(f"Отчет сохранен: {report_file}")
        
        return self.results['failed'] == 0


class LargeDataTest(BaseTest):
    """Тест работы с большими объемами данных"""
    
    def __init__(self, base_url: str, logger: logging.Logger, report_dir: Path):
        super().__init__(base_url, logger, report_dir)
        self.monitoring_data = []
    
    def monitor_resources(self, process_name: str, duration: int = 60, interval: int = 5):
        """Мониторинг ресурсов процесса"""
        if not PSUTIL_AVAILABLE:
            self.logger.warning("psutil не установлен, мониторинг ресурсов недоступен")
            return
        
        self.logger.info(f"Мониторинг ресурсов процесса '{process_name}' в течение {duration}с...")
        
        end_time = time.time() + duration
        while time.time() < end_time:
            for proc in psutil.process_iter(['pid', 'name', 'cpu_percent', 'memory_info']):
                try:
                    if process_name.lower() in proc.info['name'].lower():
                        cpu = proc.info['cpu_percent']
                        mem = proc.info['memory_info']
                        rss_mb = mem.rss / 1024 / 1024
                        vms_mb = mem.vms / 1024 / 1024
                        
                        self.monitoring_data.append({
                            'timestamp': datetime.now().isoformat(),
                            'cpu_percent': cpu,
                            'memory_mb': rss_mb,
                            'vms_mb': vms_mb
                        })
                        
                        self.logger.info(f"CPU: {cpu:.1f}%, Memory: {rss_mb:.2f}MB (RSS)")
                        break
                except (psutil.NoSuchProcess, psutil.AccessDenied):
                    continue
            
            time.sleep(interval)
    
    def run(self) -> bool:
        """Запуск теста"""
        self.logger.info("=== Тест: Работа с большими объемами данных ===")
        
        # Определяем имя процесса
        process_name = "httpserver"
        if PSUTIL_AVAILABLE:
            for proc in psutil.process_iter(['name']):
                if 'httpserver' in proc.info['name'].lower():
                    process_name = proc.info['name']
                    break
        
        # Базовый мониторинг
        self.logger.info("Базовый мониторинг ресурсов (60 секунд)...")
        monitor_thread = threading.Thread(
            target=self.monitor_resources,
            args=(process_name, 60, 5),
            daemon=True
        )
        monitor_thread.start()
        monitor_thread.join(timeout=65)
        
        # Анализ данных мониторинга
        if self.monitoring_data:
            first_mem = self.monitoring_data[0]['memory_mb']
            last_mem = self.monitoring_data[-1]['memory_mb']
            memory_increase = last_mem - first_mem
            
            if memory_increase > 100:
                self.logger.warning(f"Возможна утечка памяти: рост на {memory_increase:.2f}MB")
                self.results['warnings'] += 1
            else:
                self.logger.info(f"Рост памяти в норме: {memory_increase:.2f}MB")
            
            max_cpu = max(d['cpu_percent'] for d in self.monitoring_data)
            if max_cpu > 90:
                self.logger.warning(f"Высокая нагрузка CPU: максимум {max_cpu:.1f}%")
                self.results['warnings'] += 1
        
        # Генерируем отчет
        description = """
1. Базовый мониторинг ресурсов (60 секунд)
2. Анализ использования CPU и памяти
3. Проверка на утечки памяти
        """
        
        report_file = self.generate_report("large_data", description)
        
        # Сохраняем данные мониторинга
        if self.monitoring_data:
            csv_file = self.report_dir / f"resources_{datetime.now().strftime('%Y%m%d_%H%M%S')}.csv"
            with open(csv_file, 'w', encoding='utf-8') as f:
                f.write("timestamp,cpu_percent,memory_mb,vms_mb\n")
                for data in self.monitoring_data:
                    f.write(f"{data['timestamp']},{data['cpu_percent']},{data['memory_mb']},{data['vms_mb']}\n")
            self.logger.info(f"Данные мониторинга сохранены: {csv_file}")
        
        self.logger.info(f"Отчет сохранен: {report_file}")
        
        return True


def main():
    """Главная функция"""
    parser = argparse.ArgumentParser(description='Chaos Monkey Backend Testing')
    parser.add_argument('--test', choices=['concurrent_config', 'invalid_normalization',
                                          'ai_failure', 'large_data', 'all'],
                       default='all', help='Тест для запуска')
    parser.add_argument('--quick', action='store_true',
                       help='Быстрый режим (меньше запросов, меньше времени)')
    parser.add_argument('--base-url', 
                       default=os.getenv('CHAOS_BASE_URL', 'http://localhost:9999'),
                       help='Базовый URL сервера (по умолчанию: http://localhost:9999)')
    parser.add_argument('--log-dir', default='./chaos_tests/logs',
                       help='Директория для логов')
    parser.add_argument('--report-dir', default='./chaos_tests/reports',
                       help='Директория для отчетов')
    
    args = parser.parse_args()
    
    # Настройка логирования
    log_dir = Path(args.log_dir)
    logger = setup_logging(log_dir)
    
    logger.info("=" * 60)
    logger.info("Chaos Monkey Backend Testing - Python Version")
    logger.info("=" * 60)
    
    report_dir = Path(args.report_dir)
    report_dir.mkdir(parents=True, exist_ok=True)
    
    # Список тестов для запуска
    tests_to_run = []
    if args.test == 'all':
        tests_to_run = ['concurrent_config', 'invalid_normalization', 'ai_failure', 'large_data']
    else:
        tests_to_run = [args.test]
    
    results = {}
    
    # Запуск тестов
    for test_name in tests_to_run:
        logger.info(f"\n{'=' * 60}")
        logger.info(f"Запуск теста: {test_name}")
        logger.info(f"{'=' * 60}\n")
        
        try:
            if test_name == 'concurrent_config':
                test = ConcurrentConfigTest(args.base_url, logger, report_dir)
                if args.quick:
                    test.quick_mode = True
            elif test_name == 'invalid_normalization':
                test = InvalidNormalizationTest(args.base_url, logger, report_dir)
            elif test_name == 'ai_failure':
                test = AIFailureTest(args.base_url, logger, report_dir)
            elif test_name == 'large_data':
                test = LargeDataTest(args.base_url, logger, report_dir)
            else:
                logger.error(f"Неизвестный тест: {test_name}")
                continue
            
            success = test.run()
            results[test_name] = success
            
        except Exception as e:
            logger.error(f"Ошибка при выполнении теста {test_name}: {e}", exc_info=True)
            results[test_name] = False
    
    # Итоговый отчет
    logger.info(f"\n{'=' * 60}")
    logger.info("Итоги тестирования")
    logger.info(f"{'=' * 60}\n")
    
    total_passed = sum(1 for v in results.values() if v)
    total_failed = len(results) - total_passed
    
    logger.info(f"Всего тестов: {len(results)}")
    logger.info(f"Пройдено: {total_passed}")
    logger.info(f"Провалено: {total_failed}")
    
    # Генерация сводного отчета
    summary_file = report_dir / f"chaos_test_summary_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
    with open(summary_file, 'w', encoding='utf-8') as f:
        f.write("# Chaos Monkey Backend Testing - Сводный отчет\n\n")
        f.write(f"**Дата:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
        f.write(f"**Сервер:** {args.base_url}\n\n")
        f.write("## Результаты тестов\n\n")
        
        for test_name, success in results.items():
            status = "✅ PASSED" if success else "❌ FAILED"
            f.write(f"- **{test_name}**: {status}\n")
        
        f.write(f"\n**Всего:** {len(results)} | **Пройдено:** {total_passed} | **Провалено:** {total_failed}\n")
    
    logger.info(f"\nСводный отчет: {summary_file}")
    
    sys.exit(0 if total_failed == 0 else 1)


if __name__ == '__main__':
    main()

