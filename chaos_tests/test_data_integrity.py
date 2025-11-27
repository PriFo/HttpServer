#!/usr/bin/env python3
"""
Тесты целостности данных - проверка консистентности после операций
"""

import sys
import requests
import time
import json
from typing import Dict, List, Optional


class DataIntegrityTest:
    """Тесты целостности данных"""
    
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.session.timeout = 30
        self.issues = []
    
    def check_config_consistency(self) -> bool:
        """Проверка консистентности конфигурации"""
        print("Проверка консистентности конфигурации...")
        
        # Получаем конфигурацию
        try:
            response = self.session.get(f"{self.base_url}/api/config")
            if response.status_code != 200:
                self.issues.append(f"Не удалось получить конфигурацию: HTTP {response.status_code}")
                return False
            
            config = response.json()
            
            # Проверяем историю
            history_response = self.session.get(f"{self.base_url}/api/config/history?limit=10")
            if history_response.status_code == 200:
                history = history_response.json().get('history', [])
                current_version = history_response.json().get('current_version', 0)
                
                # Проверяем, что версии последовательны
                versions = [h.get('version') for h in history if h.get('version')]
                if versions:
                    versions.sort()
                    for i in range(1, len(versions)):
                        if versions[i] - versions[i-1] > 1:
                            self.issues.append(
                                f"Пропуск версий в истории: {versions[i-1]} -> {versions[i]}"
                            )
            
            # Проверяем, что конфигурация валидна
            required_fields = ['port', 'database_path']
            for field in required_fields:
                if field not in config:
                    self.issues.append(f"Отсутствует обязательное поле: {field}")
            
            return len(self.issues) == 0
            
        except Exception as e:
            self.issues.append(f"Ошибка при проверке конфигурации: {e}")
            return False
    
    def check_database_consistency(self) -> bool:
        """Проверка консистентности данных БД"""
        print("Проверка консистентности данных БД...")
        
        try:
            # Получаем информацию о БД
            response = self.session.get(f"{self.base_url}/api/database/info")
            if response.status_code != 200:
                self.issues.append(f"Не удалось получить информацию о БД: HTTP {response.status_code}")
                return False
            
            db_info = response.json()
            
            # Проверяем базовые метрики
            if 'total_records' in db_info:
                total = db_info['total_records']
                if total < 0:
                    self.issues.append(f"Некорректное количество записей: {total}")
            
            # Проверяем список баз данных
            db_list_response = self.session.get(f"{self.base_url}/api/databases/list")
            if db_list_response.status_code == 200:
                databases = db_list_response.json()
                if isinstance(databases, list):
                    if len(databases) == 0:
                        self.issues.append("Список баз данных пуст")
            
            return len(self.issues) == 0
            
        except Exception as e:
            self.issues.append(f"Ошибка при проверке БД: {e}")
            return False
    
    def check_normalization_consistency(self) -> bool:
        """Проверка консистентности данных нормализации"""
        print("Проверка консистентности данных нормализации...")
        
        try:
            # Получаем статус нормализации
            response = self.session.get(f"{self.base_url}/api/normalization/status")
            if response.status_code == 200:
                status = response.json()
                
                # Проверяем, что статус валиден
                valid_statuses = ['idle', 'running', 'completed', 'failed']
                if 'status' in status:
                    if status['status'] not in valid_statuses:
                        self.issues.append(f"Неизвестный статус нормализации: {status['status']}")
            
            # Получаем статистику
            stats_response = self.session.get(f"{self.base_url}/api/normalization/stats")
            if stats_response.status_code == 200:
                stats = stats_response.json()
                
                # Проверяем, что статистика неотрицательна
                numeric_fields = ['total', 'normalized', 'pending']
                for field in numeric_fields:
                    if field in stats:
                        value = stats[field]
                        if isinstance(value, (int, float)) and value < 0:
                            self.issues.append(f"Отрицательное значение в статистике: {field} = {value}")
            
            return len(self.issues) == 0
            
        except Exception as e:
            self.issues.append(f"Ошибка при проверке нормализации: {e}")
            return False
    
    def run_all_checks(self) -> Dict:
        """Запуск всех проверок"""
        print("=" * 60)
        print("Тесты целостности данных")
        print("=" * 60)
        print()
        
        results = {
            'config': self.check_config_consistency(),
            'database': self.check_database_consistency(),
            'normalization': self.check_normalization_consistency(),
            'issues': self.issues.copy()
        }
        
        print("\n" + "=" * 60)
        print("Результаты")
        print("=" * 60)
        print(f"Конфигурация: {'✅' if results['config'] else '❌'}")
        print(f"База данных: {'✅' if results['database'] else '❌'}")
        print(f"Нормализация: {'✅' if results['normalization'] else '❌'}")
        
        if results['issues']:
            print(f"\nНайдено проблем: {len(results['issues'])}")
            for issue in results['issues']:
                print(f"  - {issue}")
        else:
            print("\n✅ Проблем не обнаружено")
        
        return results


def main():
    base_url = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:9999"
    
    test = DataIntegrityTest(base_url)
    results = test.run_all_checks()
    
    # Сохраняем отчет
    from pathlib import Path
    report_dir = Path("reports")
    report_dir.mkdir(exist_ok=True)
    
    report_file = report_dir / f"data_integrity_{int(time.time())}.json"
    with open(report_file, 'w', encoding='utf-8') as f:
        json.dump(results, f, indent=2, ensure_ascii=False)
    
    print(f"\nОтчет сохранен: {report_file}")
    
    return 0 if len(results['issues']) == 0 else 1


if __name__ == "__main__":
    sys.exit(main())

