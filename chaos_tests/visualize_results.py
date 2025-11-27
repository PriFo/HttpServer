#!/usr/bin/env python3
"""
Визуализация результатов Chaos Monkey тестов
Создает графики и HTML отчеты
"""

import json
import sys
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional
from collections import defaultdict

try:
    import matplotlib
    matplotlib.use('Agg')  # Для работы без GUI
    import matplotlib.pyplot as plt
    import matplotlib.dates as mdates
    MATPLOTLIB_AVAILABLE = True
except ImportError:
    MATPLOTLIB_AVAILABLE = False
    print("Warning: matplotlib not available. Graphs will not be generated.")


class ResultsVisualizer:
    """Визуализатор результатов тестов"""
    
    def __init__(self, reports_dir: Path, output_dir: Path):
        self.reports_dir = Path(reports_dir)
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)
    
    def parse_reports(self) -> List[Dict]:
        """Парсинг всех отчетов"""
        import re
        
        reports = []
        for report_file in sorted(self.reports_dir.glob("chaos_test_summary_*.md")):
            try:
                content = report_file.read_text(encoding='utf-8')
                
                # Извлекаем дату
                date_match = re.search(r'\*\*Дата:\*\* (.+)', content)
                date_str = date_match.group(1) if date_match else None
                
                # Извлекаем результаты
                results = {}
                for line in content.split('\n'):
                    if '**' in line and ('PASSED' in line or 'FAILED' in line):
                        match = re.search(r'\*\*([^:]+):\*\* (✅|❌) (PASSED|FAILED)', line)
                        if match:
                            test_name = match.group(1).strip()
                            status = match.group(3)
                            results[test_name] = status == 'PASSED'
                
                if date_str and results:
                    try:
                        date_obj = datetime.strptime(date_str, '%Y-%m-%d %H:%M:%S')
                        reports.append({
                            'date': date_obj,
                            'results': results,
                            'file': report_file.name
                        })
                    except ValueError:
                        continue
            except Exception as e:
                print(f"Error parsing {report_file}: {e}")
                continue
        
        return sorted(reports, key=lambda x: x['date'])
    
    def create_success_rate_chart(self, reports: List[Dict]) -> Optional[Path]:
        """Создание графика успешности тестов"""
        if not MATPLOTLIB_AVAILABLE:
            return None
        
        # Подготовка данных
        test_names = set()
        for report in reports:
            test_names.update(report['results'].keys())
        
        test_names = sorted(test_names)
        dates = [r['date'] for r in reports]
        
        # Создание графика
        fig, ax = plt.subplots(figsize=(12, 6))
        
        for test_name in test_names:
            success_rates = []
            for report in reports:
                if test_name in report['results']:
                    success_rates.append(100 if report['results'][test_name] else 0)
                else:
                    success_rates.append(None)
            
            # Фильтруем None значения
            valid_dates = []
            valid_rates = []
            for i, rate in enumerate(success_rates):
                if rate is not None:
                    valid_dates.append(dates[i])
                    valid_rates.append(rate)
            
            if valid_dates:
                ax.plot(valid_dates, valid_rates, marker='o', label=test_name, linewidth=2)
        
        ax.set_xlabel('Дата', fontsize=12)
        ax.set_ylabel('Успешность (%)', fontsize=12)
        ax.set_title('Успешность тестов по времени', fontsize=14, fontweight='bold')
        ax.legend(loc='best')
        ax.grid(True, alpha=0.3)
        ax.set_ylim([-5, 105])
        
        # Форматирование дат
        ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y-%m-%d %H:%M'))
        ax.xaxis.set_major_locator(mdates.HourLocator(interval=1))
        plt.xticks(rotation=45, ha='right')
        
        plt.tight_layout()
        
        output_file = self.output_dir / 'success_rate_chart.png'
        plt.savefig(output_file, dpi=150, bbox_inches='tight')
        plt.close()
        
        return output_file
    
    def create_test_statistics_chart(self, reports: List[Dict]) -> Optional[Path]:
        """Создание графика статистики по тестам"""
        if not MATPLOTLIB_AVAILABLE:
            return None
        
        # Подсчет статистики
        test_stats = defaultdict(lambda: {'passed': 0, 'failed': 0})
        
        for report in reports:
            for test_name, passed in report['results'].items():
                if passed:
                    test_stats[test_name]['passed'] += 1
                else:
                    test_stats[test_name]['failed'] += 1
        
        # Подготовка данных
        test_names = list(test_stats.keys())
        passed_counts = [test_stats[t]['passed'] for t in test_names]
        failed_counts = [test_stats[t]['failed'] for t in test_names]
        
        # Создание графика
        fig, ax = plt.subplots(figsize=(10, 6))
        
        x = range(len(test_names))
        width = 0.35
        
        bars1 = ax.bar([i - width/2 for i in x], passed_counts, width, label='Пройдено', color='#2ecc71')
        bars2 = ax.bar([i + width/2 for i in x], failed_counts, width, label='Провалено', color='#e74c3c')
        
        ax.set_xlabel('Тесты', fontsize=12)
        ax.set_ylabel('Количество', fontsize=12)
        ax.set_title('Статистика выполнения тестов', fontsize=14, fontweight='bold')
        ax.set_xticks(x)
        ax.set_xticklabels(test_names, rotation=45, ha='right')
        ax.legend()
        ax.grid(True, alpha=0.3, axis='y')
        
        # Добавляем значения на столбцы
        for bars in [bars1, bars2]:
            for bar in bars:
                height = bar.get_height()
                if height > 0:
                    ax.text(bar.get_x() + bar.get_width()/2., height,
                           f'{int(height)}',
                           ha='center', va='bottom', fontsize=9)
        
        plt.tight_layout()
        
        output_file = self.output_dir / 'test_statistics_chart.png'
        plt.savefig(output_file, dpi=150, bbox_inches='tight')
        plt.close()
        
        return output_file
    
    def create_html_dashboard(self, reports: List[Dict]) -> Path:
        """Создание HTML дашборда"""
        html_content = f"""
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chaos Monkey Test Dashboard</title>
    <style>
        body {{
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }}
        .container {{
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }}
        h1 {{
            color: #2c3e50;
            border-bottom: 3px solid #3498db;
            padding-bottom: 10px;
        }}
        .stats {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 30px 0;
        }}
        .stat-card {{
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }}
        .stat-card h3 {{
            margin: 0 0 10px 0;
            font-size: 14px;
            opacity: 0.9;
        }}
        .stat-card .value {{
            font-size: 32px;
            font-weight: bold;
        }}
        .test-results {{
            margin-top: 30px;
        }}
        .test-item {{
            background: #f8f9fa;
            padding: 15px;
            margin: 10px 0;
            border-radius: 5px;
            border-left: 4px solid #3498db;
        }}
        .test-item.passed {{
            border-left-color: #2ecc71;
        }}
        .test-item.failed {{
            border-left-color: #e74c3c;
        }}
        .test-name {{
            font-weight: bold;
            font-size: 16px;
            color: #2c3e50;
        }}
        .test-status {{
            display: inline-block;
            padding: 5px 10px;
            border-radius: 3px;
            font-size: 12px;
            margin-left: 10px;
        }}
        .status-passed {{
            background: #2ecc71;
            color: white;
        }}
        .status-failed {{
            background: #e74c3c;
            color: white;
        }}
        .chart-container {{
            margin: 30px 0;
            text-align: center;
        }}
        .chart-container img {{
            max-width: 100%;
            height: auto;
            border: 1px solid #ddd;
            border-radius: 5px;
        }}
        table {{
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }}
        th, td {{
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }}
        th {{
            background: #3498db;
            color: white;
        }}
        tr:hover {{
            background: #f5f5f5;
        }}
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Chaos Monkey Test Dashboard</h1>
        <p><strong>Дата генерации:</strong> {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
        
        <div class="stats">
            <div class="stat-card">
                <h3>Всего отчетов</h3>
                <div class="value">{len(reports)}</div>
            </div>
            <div class="stat-card">
                <h3>Последний запуск</h3>
                <div class="value">{reports[-1]['date'].strftime('%H:%M') if reports else 'N/A'}</div>
            </div>
        </div>
"""
        
        # Статистика по тестам
        test_stats = defaultdict(lambda: {'passed': 0, 'failed': 0})
        for report in reports:
            for test_name, passed in report['results'].items():
                if passed:
                    test_stats[test_name]['passed'] += 1
                else:
                    test_stats[test_name]['failed'] += 1
        
        html_content += """
        <h2>Статистика по тестам</h2>
        <table>
            <thead>
                <tr>
                    <th>Тест</th>
                    <th>Пройдено</th>
                    <th>Провалено</th>
                    <th>Успешность</th>
                </tr>
            </thead>
            <tbody>
"""
        
        for test_name in sorted(test_stats.keys()):
            stats = test_stats[test_name]
            total = stats['passed'] + stats['failed']
            success_rate = (stats['passed'] / total * 100) if total > 0 else 0
            
            html_content += f"""
                <tr>
                    <td><strong>{test_name}</strong></td>
                    <td>{stats['passed']}</td>
                    <td>{stats['failed']}</td>
                    <td>{success_rate:.1f}%</td>
                </tr>
"""
        
        html_content += """
            </tbody>
        </table>
"""
        
        # Графики
        if MATPLOTLIB_AVAILABLE:
            chart1 = self.create_success_rate_chart(reports)
            chart2 = self.create_test_statistics_chart(reports)
            
            if chart1 and chart1.exists():
                html_content += f"""
        <div class="chart-container">
            <h2>График успешности тестов</h2>
            <img src="{chart1.name}" alt="Success Rate Chart">
        </div>
"""
            
            if chart2 and chart2.exists():
                html_content += f"""
        <div class="chart-container">
            <h2>Статистика выполнения тестов</h2>
            <img src="{chart2.name}" alt="Test Statistics Chart">
        </div>
"""
        
        # Последние результаты
        html_content += """
        <h2>Последние результаты</h2>
        <div class="test-results">
"""
        
        for report in reports[-10:]:  # Последние 10 отчетов
            html_content += f"""
            <div class="test-item">
                <div class="test-name">
                    {report['date'].strftime('%Y-%m-%d %H:%M:%S')}
                </div>
"""
            for test_name, passed in report['results'].items():
                status_class = 'passed' if passed else 'failed'
                status_text = 'PASSED' if passed else 'FAILED'
                status_icon = '✅' if passed else '❌'
                html_content += f"""
                <div style="margin: 5px 0;">
                    {status_icon} <strong>{test_name}</strong>
                    <span class="test-status status-{status_class}">{status_text}</span>
                </div>
"""
            html_content += """
            </div>
"""
        
        html_content += """
        </div>
    </div>
</body>
</html>
"""
        
        output_file = self.output_dir / 'dashboard.html'
        output_file.write_text(html_content, encoding='utf-8')
        
        return output_file


def main():
    """Главная функция"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Визуализация результатов Chaos Monkey тестов')
    parser.add_argument('--reports-dir', type=str, default='./reports',
                       help='Директория с отчетами')
    parser.add_argument('--output-dir', type=str, default='./reports',
                       help='Директория для сохранения визуализаций')
    
    args = parser.parse_args()
    
    visualizer = ResultsVisualizer(args.reports_dir, args.output_dir)
    reports = visualizer.parse_reports()
    
    if not reports:
        print("❌ Отчеты не найдены")
        return
    
    print(f"✅ Найдено отчетов: {len(reports)}")
    
    # Создание дашборда
    dashboard_file = visualizer.create_html_dashboard(reports)
    print(f"✅ HTML дашборд создан: {dashboard_file}")
    
    if MATPLOTLIB_AVAILABLE:
        chart1 = visualizer.create_success_rate_chart(reports)
        if chart1:
            print(f"✅ График успешности создан: {chart1}")
        
        chart2 = visualizer.create_test_statistics_chart(reports)
        if chart2:
            print(f"✅ График статистики создан: {chart2}")
    else:
        print("⚠️ matplotlib не установлен, графики не созданы")


if __name__ == '__main__':
    main()

