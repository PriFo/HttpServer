#!/usr/bin/env python3
"""
Интегрированная версия Chaos Monkey с новыми тестами
Объединяет chaos_monkey.py и improved_tests.py
"""

import sys
from pathlib import Path

# Добавляем текущую директорию в путь
sys.path.insert(0, str(Path(__file__).parent))

# Импортируем основные классы
from chaos_monkey import (
    BaseTest, ConcurrentConfigTest, InvalidNormalizationTest,
    AIFailureTest, LargeDataTest, setup_logging, main as chaos_main
)

# Импортируем улучшенные тесты
from improved_tests import DatabaseLockTest, StressTest, ResourceMonitor


class IntegratedConcurrentConfigTest(ConcurrentConfigTest):
    """Расширенный тест конкурентных обновлений с мониторингом ресурсов"""
    
    def __init__(self, base_url: str, logger, report_dir: Path):
        super().__init__(base_url, logger, report_dir)
        self.resource_monitor = None
    
    def run(self) -> bool:
        """Запуск с мониторингом ресурсов"""
        if ResourceMonitor and hasattr(ResourceMonitor, 'start_monitoring'):
            self.resource_monitor = ResourceMonitor(self.logger)
            self.resource_monitor.start_monitoring()
        
        try:
            result = super().run()
            return result
        finally:
            if self.resource_monitor:
                stats = self.resource_monitor.stop_monitoring()
                if stats and 'error' not in stats:
                    self.logger.info(f"Ресурсы: CPU avg={stats['cpu']['avg']:.1f}%, "
                                   f"Memory avg={stats['memory']['avg_mb']:.1f}MB")


def main():
    """Главная функция с поддержкой новых тестов"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Integrated Chaos Monkey Testing')
    parser.add_argument('--test', type=str, default='all',
                       choices=['all', 'concurrent_config', 'invalid_normalization',
                               'ai_failure', 'large_data', 'db_lock', 'stress'],
                       help='Тест для запуска')
    parser.add_argument('--base-url', type=str, default='http://localhost:9999',
                       help='Базовый URL сервера')
    parser.add_argument('--quick', action='store_true',
                       help='Быстрый режим')
    parser.add_argument('--report-dir', type=str, default='./reports',
                       help='Директория для отчетов')
    
    args = parser.parse_args()
    
    # Настройка логирования
    log_dir = Path('./logs')
    logger = setup_logging(log_dir)
    report_dir = Path(args.report_dir)
    report_dir.mkdir(parents=True, exist_ok=True)
    
    # Определяем тесты для запуска
    if args.test == 'all':
        tests_to_run = ['concurrent_config', 'invalid_normalization', 'ai_failure', 'large_data']
    else:
        tests_to_run = [args.test]
    
    results = {}
    
    # Запуск стандартных тестов
    for test_name in tests_to_run:
        if test_name in ['db_lock', 'stress']:
            continue  # Пропускаем новые тесты, обработаем отдельно
        
        logger.info(f"\n{'=' * 60}")
        logger.info(f"Запуск теста: {test_name}")
        logger.info(f"{'=' * 60}\n")
        
        try:
            if test_name == 'concurrent_config':
                test = IntegratedConcurrentConfigTest(args.base_url, logger, report_dir)
                if args.quick:
                    test.quick_mode = True
            elif test_name == 'invalid_normalization':
                test = InvalidNormalizationTest(args.base_url, logger, report_dir)
            elif test_name == 'ai_failure':
                test = AIFailureTest(args.base_url, logger, report_dir)
            elif test_name == 'large_data':
                test = LargeDataTest(args.base_url, logger, report_dir)
            else:
                continue
            
            success = test.run()
            results[test_name] = success
            
        except Exception as e:
            logger.error(f"Ошибка при выполнении теста {test_name}: {e}", exc_info=True)
            results[test_name] = False
    
    # Запуск новых тестов
    if args.test in ['all', 'db_lock', 'stress']:
        if args.test == 'all' or args.test == 'db_lock':
            logger.info(f"\n{'=' * 60}")
            logger.info("Запуск теста: db_lock (Database Lock Test)")
            logger.info(f"{'=' * 60}\n")
            
            try:
                db_test = DatabaseLockTest(args.base_url, logger)
                db_results = db_test.run_concurrent_updates(num_threads=20)
                
                logger.info(f"Результаты DB Lock Test:")
                logger.info(f"  Успешно: {db_results['success']}")
                logger.info(f"  Провалено: {db_results['failed']}")
                logger.info(f"  Блокировки БД: {db_results['database_locked']}")
                
                results['db_lock'] = db_results['database_locked'] == 0
                
            except Exception as e:
                logger.error(f"Ошибка при выполнении db_lock теста: {e}", exc_info=True)
                results['db_lock'] = False
        
        if args.test == 'all' or args.test == 'stress':
            logger.info(f"\n{'=' * 60}")
            logger.info("Запуск теста: stress (Stress Test)")
            logger.info(f"{'=' * 60}\n")
            
            try:
                stress_test = StressTest(args.base_url, logger)
                stress_results = stress_test.stress_endpoint(
                    '/api/config',
                    method='GET',
                    num_requests=100,
                    concurrency=10
                )
                
                logger.info(f"Результаты Stress Test:")
                logger.info(f"  Успешно: {stress_results['success']}")
                logger.info(f"  Провалено: {stress_results['failed']}")
                logger.info(f"  Среднее время ответа: {stress_results.get('avg_response_time', 0):.3f}s")
                logger.info(f"  Запросов в секунду: {stress_results.get('requests_per_second', 0):.2f}")
                
                results['stress'] = stress_results['failed'] < stress_results['total'] * 0.1  # Менее 10% провалов
                
            except Exception as e:
                logger.error(f"Ошибка при выполнении stress теста: {e}", exc_info=True)
                results['stress'] = False
    
    # Итоговый отчет
    logger.info(f"\n{'=' * 60}")
    logger.info("Итоги тестирования")
    logger.info(f"{'=' * 60}\n")
    
    total_passed = sum(1 for v in results.values() if v)
    total_failed = len(results) - total_passed
    
    logger.info(f"Всего тестов: {len(results)}")
    logger.info(f"Пройдено: {total_passed}")
    logger.info(f"Провалено: {total_failed}")
    
    # Используем стандартную функцию генерации отчета из chaos_monkey
    from chaos_monkey import datetime
    summary_file = report_dir / f"chaos_test_summary_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
    
    with open(summary_file, 'w', encoding='utf-8') as f:
        f.write("# Chaos Monkey Backend Testing - Сводный отчет (Integrated)\n\n")
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

