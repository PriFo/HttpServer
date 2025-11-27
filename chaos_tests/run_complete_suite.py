#!/usr/bin/env python3
"""
Полный набор тестов Chaos Monkey
Запускает все тесты, анализирует результаты и создает визуализацию
"""

import sys
import subprocess
from pathlib import Path
from datetime import datetime


def run_command(cmd: list, description: str) -> bool:
    """Запуск команды с обработкой ошибок"""
    print(f"\n{'=' * 60}")
    print(f"{description}")
    print(f"{'=' * 60}\n")
    
    try:
        result = subprocess.run(cmd, check=False, capture_output=False)
        return result.returncode == 0
    except Exception as e:
        print(f"❌ Ошибка: {e}")
        return False


def main():
    """Главная функция"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Полный набор тестов Chaos Monkey')
    parser.add_argument('--base-url', type=str, default='http://localhost:9999',
                       help='Базовый URL сервера')
    parser.add_argument('--auto-start', action='store_true',
                       help='Автоматически запустить сервер')
    parser.add_argument('--quick', action='store_true',
                       help='Быстрый режим')
    parser.add_argument('--skip-visualization', action='store_true',
                       help='Пропустить создание визуализации')
    
    args = parser.parse_args()
    
    script_dir = Path(__file__).parent
    reports_dir = script_dir / 'reports'
    reports_dir.mkdir(parents=True, exist_ok=True)
    
    print("=" * 60)
    print("🚀 Chaos Monkey - Полный набор тестов")
    print("=" * 60)
    print(f"Дата: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"Сервер: {args.base_url}")
    print("=" * 60)
    
    results = {}
    
    # 1. Проверка/запуск сервера
    if args.auto_start:
        print("\n📡 Проверка и запуск сервера...")
        from test_runner import check_server, start_server_if_needed, wait_for_server
        
        if not check_server(args.base_url, max_attempts=1):
            if start_server_if_needed():
                if not wait_for_server(args.base_url, timeout=60):
                    print("❌ Сервер не запустился")
                    sys.exit(1)
            else:
                print("❌ Не удалось запустить сервер")
                sys.exit(1)
        print("✅ Сервер готов")
    
    # 2. Запуск интегрированных тестов
    print("\n🧪 Запуск интегрированных тестов...")
    test_cmd = [sys.executable, str(script_dir / 'integrated_chaos_monkey.py'),
                '--test', 'all', '--base-url', args.base_url]
    if args.quick:
        test_cmd.append('--quick')
    
    results['tests'] = run_command(test_cmd, "Интегрированные тесты")
    
    # 3. Запуск улучшенных тестов
    print("\n🔬 Запуск улучшенных тестов...")
    improved_tests = ['db_lock', 'stress']
    for test_name in improved_tests:
        cmd = [sys.executable, str(script_dir / 'improved_tests.py'),
               '--test', test_name, '--base-url', args.base_url]
        results[f'improved_{test_name}'] = run_command(cmd, f"Улучшенный тест: {test_name}")
    
    # 4. Анализ отчетов
    print("\n📊 Анализ отчетов...")
    analyze_cmd = [sys.executable, str(script_dir / 'report_analyzer.py'),
                   '--reports-dir', str(reports_dir)]
    results['analysis'] = run_command(analyze_cmd, "Анализ отчетов")
    
    # 5. Визуализация результатов
    if not args.skip_visualization:
        print("\n📈 Создание визуализации...")
        viz_cmd = [sys.executable, str(script_dir / 'visualize_results.py'),
                   '--reports-dir', str(reports_dir),
                   '--output-dir', str(reports_dir)]
        results['visualization'] = run_command(viz_cmd, "Визуализация результатов")
    
    # Итоги
    print("\n" + "=" * 60)
    print("📋 Итоги выполнения")
    print("=" * 60)
    
    total = len(results)
    passed = sum(1 for v in results.values() if v)
    failed = total - passed
    
    print(f"Всего этапов: {total}")
    print(f"✅ Успешно: {passed}")
    print(f"❌ Провалено: {failed}")
    
    print("\nДетали:")
    for stage, success in results.items():
        status = "✅" if success else "❌"
        print(f"  {status} {stage}")
    
    # Открытие дашборда
    dashboard_file = reports_dir / 'dashboard.html'
    if dashboard_file.exists() and not args.skip_visualization:
        print(f"\n📊 Дашборд доступен: {dashboard_file}")
        print("Откройте в браузере для просмотра результатов")
    
    sys.exit(0 if failed == 0 else 1)


if __name__ == '__main__':
    main()

