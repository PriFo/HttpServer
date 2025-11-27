#!/usr/bin/env python3
"""
Конфигурация для Chaos Monkey тестов
Централизованное управление настройками
"""

from pathlib import Path
from typing import Dict, Optional
import json
import os


class ChaosMonkeyConfig:
    """Конфигурация Chaos Monkey тестов"""
    
    # Пути по умолчанию
    DEFAULT_BASE_URL = "http://localhost:9999"
    DEFAULT_REPORTS_DIR = "./reports"
    DEFAULT_LOGS_DIR = "./logs"
    DEFAULT_CONFIG_FILE = "./chaos_config.json"
    
    # Настройки тестов
    DEFAULT_CONCURRENT_REQUESTS = 10
    DEFAULT_QUICK_MODE_REQUESTS = 5
    DEFAULT_STRESS_REQUESTS = 100
    DEFAULT_STRESS_CONCURRENCY = 10
    DEFAULT_MONITOR_DURATION = 60
    DEFAULT_MONITOR_INTERVAL = 5
    
    # Таймауты
    DEFAULT_HTTP_TIMEOUT = 30
    DEFAULT_SERVER_WAIT_TIMEOUT = 60
    DEFAULT_RETRY_DELAY = 1
    DEFAULT_MAX_RETRIES = 3
    
    def __init__(self, config_file: Optional[str] = None):
        self.config_file = Path(config_file or self.DEFAULT_CONFIG_FILE)
        self.config = self._load_config()
    
    def _load_config(self) -> Dict:
        """Загрузка конфигурации из файла или переменных окружения"""
        config = {
            'base_url': os.getenv('CHAOS_BASE_URL', self.DEFAULT_BASE_URL),
            'reports_dir': os.getenv('CHAOS_REPORTS_DIR', self.DEFAULT_REPORTS_DIR),
            'logs_dir': os.getenv('CHAOS_LOGS_DIR', self.DEFAULT_LOGS_DIR),
            'tests': {
                'concurrent_config': {
                    'num_requests': int(os.getenv('CHAOS_CONCURRENT_REQUESTS', self.DEFAULT_CONCURRENT_REQUESTS)),
                    'quick_mode_requests': int(os.getenv('CHAOS_QUICK_REQUESTS', self.DEFAULT_QUICK_MODE_REQUESTS))
                },
                'stress': {
                    'num_requests': int(os.getenv('CHAOS_STRESS_REQUESTS', self.DEFAULT_STRESS_REQUESTS)),
                    'concurrency': int(os.getenv('CHAOS_STRESS_CONCURRENCY', self.DEFAULT_STRESS_CONCURRENCY))
                },
                'monitor': {
                    'duration': int(os.getenv('CHAOS_MONITOR_DURATION', self.DEFAULT_MONITOR_DURATION)),
                    'interval': int(os.getenv('CHAOS_MONITOR_INTERVAL', self.DEFAULT_MONITOR_INTERVAL))
                }
            },
            'timeouts': {
                'http': int(os.getenv('CHAOS_HTTP_TIMEOUT', self.DEFAULT_HTTP_TIMEOUT)),
                'server_wait': int(os.getenv('CHAOS_SERVER_WAIT_TIMEOUT', self.DEFAULT_SERVER_WAIT_TIMEOUT)),
                'retry_delay': int(os.getenv('CHAOS_RETRY_DELAY', self.DEFAULT_RETRY_DELAY)),
                'max_retries': int(os.getenv('CHAOS_MAX_RETRIES', self.DEFAULT_MAX_RETRIES))
            },
            'server': {
                'auto_start': os.getenv('CHAOS_AUTO_START', 'false').lower() == 'true',
                'api_key': os.getenv('ARLIAI_API_KEY', ''),
                'executable_path': os.getenv('CHAOS_SERVER_EXE', '')
            }
        }
        
        # Загрузка из файла, если существует
        if self.config_file.exists():
            try:
                file_config = json.loads(self.config_file.read_text(encoding='utf-8'))
                config.update(file_config)
            except Exception as e:
                print(f"Warning: Could not load config from {self.config_file}: {e}")
        
        return config
    
    def save_config(self):
        """Сохранение конфигурации в файл"""
        try:
            self.config_file.write_text(
                json.dumps(self.config, indent=2, ensure_ascii=False),
                encoding='utf-8'
            )
            return True
        except Exception as e:
            print(f"Error saving config: {e}")
            return False
    
    @property
    def base_url(self) -> str:
        return self.config['base_url']
    
    @property
    def reports_dir(self) -> Path:
        return Path(self.config['reports_dir'])
    
    @property
    def logs_dir(self) -> Path:
        return Path(self.config['logs_dir'])
    
    @property
    def concurrent_requests(self) -> int:
        return self.config['tests']['concurrent_config']['num_requests']
    
    @property
    def quick_mode_requests(self) -> int:
        return self.config['tests']['concurrent_config']['quick_mode_requests']
    
    @property
    def stress_requests(self) -> int:
        return self.config['tests']['stress']['num_requests']
    
    @property
    def stress_concurrency(self) -> int:
        return self.config['tests']['stress']['concurrency']
    
    @property
    def monitor_duration(self) -> int:
        return self.config['tests']['monitor']['duration']
    
    @property
    def monitor_interval(self) -> int:
        return self.config['tests']['monitor']['interval']
    
    @property
    def http_timeout(self) -> int:
        return self.config['timeouts']['http']
    
    @property
    def server_wait_timeout(self) -> int:
        return self.config['timeouts']['server_wait']
    
    @property
    def auto_start_server(self) -> bool:
        return self.config['server']['auto_start']
    
    @property
    def server_api_key(self) -> str:
        return self.config['server']['api_key']
    
    @property
    def server_executable(self) -> str:
        return self.config['server']['executable_path']


# Глобальный экземпляр конфигурации
_config_instance: Optional[ChaosMonkeyConfig] = None


def get_config(config_file: Optional[str] = None) -> ChaosMonkeyConfig:
    """Получение глобального экземпляра конфигурации"""
    global _config_instance
    if _config_instance is None:
        _config_instance = ChaosMonkeyConfig(config_file)
    return _config_instance


def reset_config():
    """Сброс глобального экземпляра конфигурации"""
    global _config_instance
    _config_instance = None

