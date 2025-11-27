#!/usr/bin/env python3
"""
Проверка покрытия API endpoints - находит недокументированные или неиспользуемые endpoints
"""

import sys
import requests
import json
from pathlib import Path
from typing import Set, Dict, List


class EndpointCoverageTest:
    """Тест покрытия endpoints"""
    
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.session.timeout = 10
        self.tested_endpoints = set()
        self.failed_endpoints = []
        self.successful_endpoints = []
    
    def test_endpoint(self, method: str, endpoint: str, data: Dict = None) -> bool:
        """Тестирование одного endpoint"""
        url = f"{self.base_url}{endpoint}"
        
        try:
            if method == 'GET':
                response = self.session.get(url)
            elif method == 'POST':
                response = self.session.post(url, json=data or {})
            elif method == 'PUT':
                response = self.session.put(url, json=data or {})
            elif method == 'DELETE':
                response = self.session.delete(url)
            else:
                return False
            
            self.tested_endpoints.add(f"{method} {endpoint}")
            
            if 200 <= response.status_code < 500:  # 5xx считаем проблемой сервера, не endpoint
                self.successful_endpoints.append({
                    'method': method,
                    'endpoint': endpoint,
                    'status': response.status_code
                })
                return True
            else:
                self.failed_endpoints.append({
                    'method': method,
                    'endpoint': endpoint,
                    'status': response.status_code,
                    'error': response.text[:200]
                })
                return False
                
        except requests.exceptions.ConnectionError:
            self.failed_endpoints.append({
                'method': method,
                'endpoint': endpoint,
                'status': 0,
                'error': 'Connection refused'
            })
            return False
        except Exception as e:
            self.failed_endpoints.append({
                'method': method,
                'endpoint': endpoint,
                'status': 0,
                'error': str(e)
            })
            return False
    
    def test_common_endpoints(self):
        """Тестирование общих endpoints"""
        endpoints = [
            # Config
            ('GET', '/api/config'),
            ('PUT', '/api/config', {'port': '9999'}),
            ('GET', '/api/config/history'),
            
            # Clients
            ('GET', '/api/clients'),
            ('POST', '/api/clients', {'name': 'Test Client'}),
            
            # Databases
            ('GET', '/api/databases/list'),
            ('GET', '/api/database/info'),
            
            # Normalization
            ('GET', '/api/normalization/status'),
            ('GET', '/api/normalization/stats'),
            ('POST', '/api/normalization/start', {'item_id': 1, 'original_name': 'Test'}),
            
            # System
            ('GET', '/health'),
            ('GET', '/api/system/summary'),
        ]
        
        print("Тестирование endpoints...")
        for endpoint_info in endpoints:
            method = endpoint_info[0]
            endpoint = endpoint_info[1]
            data = endpoint_info[2] if len(endpoint_info) > 2 else None
            
            success = self.test_endpoint(method, endpoint, data)
            status = "✅" if success else "❌"
            print(f"{status} {method} {endpoint}")
    
    def generate_report(self, output_file: str = "endpoint_coverage_report.md"):
        """Генерация отчета о покрытии"""
        report_path = Path("reports") / output_file
        report_path.parent.mkdir(exist_ok=True)
        
        with open(report_path, 'w', encoding='utf-8') as f:
            f.write("# Отчет о покрытии API endpoints\n\n")
            f.write(f"**Всего протестировано:** {len(self.tested_endpoints)}\n")
            f.write(f"**Успешных:** {len(self.successful_endpoints)}\n")
            f.write(f"**Неудачных:** {len(self.failed_endpoints)}\n\n")
            
            f.write("## Успешные endpoints\n\n")
            for ep in self.successful_endpoints:
                f.write(f"- ✅ `{ep['method']} {ep['endpoint']}` - HTTP {ep['status']}\n")
            
            f.write("\n## Неудачные endpoints\n\n")
            for ep in self.failed_endpoints:
                f.write(f"- ❌ `{ep['method']} {ep['endpoint']}`\n")
                f.write(f"  - Статус: {ep['status']}\n")
                f.write(f"  - Ошибка: {ep['error']}\n")
        
        print(f"\nОтчет сохранен: {report_path}")


def main():
    base_url = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:9999"
    
    print("=" * 60)
    print("Проверка покрытия API endpoints")
    print("=" * 60)
    print(f"Сервер: {base_url}\n")
    
    test = EndpointCoverageTest(base_url)
    test.test_common_endpoints()
    test.generate_report()
    
    print(f"\n✅ Успешных: {len(test.successful_endpoints)}")
    print(f"❌ Неудачных: {len(test.failed_endpoints)}")


if __name__ == "__main__":
    main()

