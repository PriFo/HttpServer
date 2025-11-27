#!/usr/bin/env python3
"""
Анализатор отчетов Chaos Monkey тестов
Парсит отчеты и создает сводную статистику
"""

import json
import re
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional
from collections import defaultdict


class ReportAnalyzer:
    """Анализатор отчетов тестов"""
    
    def __init__(self, reports_dir: Path):
        self.reports_dir = Path(reports_dir)
        self.reports_dir.mkdir(parents=True, exist_ok=True)
        self.summary_data = {
            'total_runs': 0,
            'tests': defaultdict(lambda: {
                'total': 0,
                'passed': 0,
                'failed': 0,
                'last_run': None,
                'last_status': None
            }),
            'trends': []
        }
    
    def find_reports(self) -> List[Path]:
        """Поиск всех отчетов"""
        reports = []
        
        # Ищем сводные отчеты
        for report_file in self.reports_dir.glob("chaos_test_summary_*.md"):
            reports.append(report_file)
        
        # Ищем индивидуальные отчеты
        for pattern in ["concurrent_config_*.md", "invalid_normalization_*.md", 
                       "ai_failure_*.md", "large_data_*.md"]:
            for report_file in self.reports_dir.glob(pattern):
                reports.append(report_file)
        
        return sorted(reports, reverse=True)  # Новые сначала
    
    def parse_summary_report(self, report_file: Path) -> Optional[Dict]:
        """Парсинг сводного отчета"""
        try:
            content = report_file.read_text(encoding='utf-8')
            
            # Извлекаем дату
            date_match = re.search(r'\*\*Дата:\*\* (.+)', content)
            date_str = date_match.group(1) if date_match else None
            
            # Извлекаем сервер
            server_match = re.search(r'\*\*Сервер:\*\* (.+)', content)
            server = server_match.group(1) if server_match else None
            
            # Извлекаем результаты тестов
            results = {}
            for line in content.split('\n'):
                if '**' in line and ('PASSED' in line or 'FAILED' in line):
                    # Формат: - **test_name**: ✅ PASSED или ❌ FAILED
                    match = re.search(r'\*\*([^:]+):\*\* (✅|❌) (PASSED|FAILED)', line)
                    if match:
                        test_name = match.group(1).strip()
                        status = match.group(3)
                        results[test_name] = status == 'PASSED'
            
            # Извлекаем итоги
            totals_match = re.search(r'\*\*Всего:\*\* (\d+) \| \*\*Пройдено:\*\* (\d+) \| \*\*Провалено:\*\* (\d+)', content)
            if totals_match:
                total = int(totals_match.group(1))
                passed = int(totals_match.group(2))
                failed = int(totals_match.group(3))
            else:
                total = len(results)
                passed = sum(1 for v in results.values() if v)
                failed = total - passed
            
            return {
                'file': str(report_file),
                'date': date_str,
                'server': server,
                'results': results,
                'total': total,
                'passed': passed,
                'failed': failed
            }
        except Exception as e:
            print(f"Ошибка при парсинге {report_file}: {e}")
            return None
    
    def analyze_all_reports(self) -> Dict:
        """Анализ всех отчетов"""
        reports = self.find_reports()
        
        if not reports:
            return {'error': 'No reports found'}
        
        summary_reports = [r for r in reports if 'summary' in r.name]
        
        all_results = []
        for report_file in summary_reports[:20]:  # Последние 20 отчетов
            parsed = self.parse_summary_report(report_file)
            if parsed:
                all_results.append(parsed)
        
        # Анализ трендов
        for report in all_results:
            self.summary_data['total_runs'] += 1
            for test_name, passed in report['results'].items():
                test_data = self.summary_data['tests'][test_name]
                test_data['total'] += 1
                if passed:
                    test_data['passed'] += 1
                else:
                    test_data['failed'] += 1
                test_data['last_run'] = report['date']
                test_data['last_status'] = 'PASSED' if passed else 'FAILED'
        
        return {
            'total_reports': len(all_results),
            'tests': dict(self.summary_data['tests']),
            'recent_reports': all_results[:5]  # Последние 5 отчетов
        }
    
    def generate_analysis_report(self) -> Path:
        """Генерация отчета анализа"""
        analysis = self.analyze_all_reports()
        
        if 'error' in analysis:
            print(f"Ошибка: {analysis['error']}")
            return None
        
        report_file = self.reports_dir / f"analysis_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
        
        with open(report_file, 'w', encoding='utf-8') as f:
            f.write("# Анализ отчетов Chaos Monkey тестов\n\n")
            f.write(f"**Дата анализа:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
            f.write(f"**Всего отчетов проанализировано:** {analysis['total_reports']}\n\n")
            
            f.write("## Статистика по тестам\n\n")
            f.write("| Тест | Всего запусков | Пройдено | Провалено | Успешность | Последний статус |\n")
            f.write("|------|----------------|----------|-----------|------------|-------------------|\n")
            
            for test_name, test_data in analysis['tests'].items():
                total = test_data['total']
                passed = test_data['passed']
                failed = test_data['failed']
                success_rate = (passed / total * 100) if total > 0 else 0
                last_status = test_data['last_status']
                
                f.write(f"| {test_name} | {total} | {passed} | {failed} | {success_rate:.1f}% | {last_status} |\n")
            
            f.write("\n## Последние запуски\n\n")
            for report in analysis['recent_reports']:
                f.write(f"### {report['date']}\n\n")
                f.write(f"- **Сервер:** {report['server']}\n")
                f.write(f"- **Всего:** {report['total']} | **Пройдено:** {report['passed']} | **Провалено:** {report['failed']}\n\n")
                
                for test_name, passed in report['results'].items():
                    status = "✅ PASSED" if passed else "❌ FAILED"
                    f.write(f"  - {test_name}: {status}\n")
                f.write("\n")
            
            f.write("\n## Рекомендации\n\n")
            
            # Анализ проблемных тестов
            problematic_tests = []
            for test_name, test_data in analysis['tests'].items():
                if test_data['total'] > 0:
                    failure_rate = test_data['failed'] / test_data['total']
                    if failure_rate > 0.5:  # Более 50% провалов
                        problematic_tests.append((test_name, failure_rate))
            
            if problematic_tests:
                f.write("### Проблемные тесты (более 50% провалов):\n\n")
                for test_name, failure_rate in sorted(problematic_tests, key=lambda x: x[1], reverse=True):
                    f.write(f"- **{test_name}**: {failure_rate*100:.1f}% провалов\n")
                f.write("\n")
            else:
                f.write("✅ Все тесты показывают стабильные результаты\n\n")
        
        return report_file


def main():
    """Главная функция"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Анализатор отчетов Chaos Monkey')
    parser.add_argument('--reports-dir', type=str, default='./reports',
                       help='Директория с отчетами')
    parser.add_argument('--output', type=str, default=None,
                       help='Файл для сохранения анализа (по умолчанию: reports/analysis_*.md)')
    
    args = parser.parse_args()
    
    analyzer = ReportAnalyzer(args.reports_dir)
    report_file = analyzer.generate_analysis_report()
    
    if report_file:
        print(f"✅ Анализ завершен: {report_file}")
    else:
        print("❌ Ошибка при создании анализа")


if __name__ == '__main__':
    main()

