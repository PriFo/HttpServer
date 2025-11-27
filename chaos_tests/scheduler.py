#!/usr/bin/env python3
"""
Планировщик тестов Chaos Monkey
Запуск тестов по расписанию
"""

import sys
import time
import subprocess
from pathlib import Path
from datetime import datetime, timedelta
from typing import Dict, List, Optional
import json


class TestScheduler:
    """Планировщик тестов"""
    
    def __init__(self, schedule_file: Path):
        self.schedule_file = Path(schedule_file)
        self.schedule_file.parent.mkdir(parents=True, exist_ok=True)
        self.schedule = self._load_schedule()
    
    def _load_schedule(self) -> Dict:
        """Загрузка расписания из файла"""
        if self.schedule_file.exists():
            try:
                return json.loads(self.schedule_file.read_text(encoding='utf-8'))
            except:
                return {}
        return {}
    
    def _save_schedule(self):
        """Сохранение расписания в файл"""
        self.schedule_file.write_text(
            json.dumps(self.schedule, indent=2, ensure_ascii=False),
            encoding='utf-8'
        )
    
    def add_schedule(self, name: str, cron_expression: str, command: List[str], enabled: bool = True):
        """Добавление задачи в расписание"""
        self.schedule[name] = {
            'cron': cron_expression,
            'command': command,
            'enabled': enabled,
            'last_run': None,
            'next_run': None
        }
        self._save_schedule()
    
    def remove_schedule(self, name: str):
        """Удаление задачи из расписания"""
        if name in self.schedule:
            del self.schedule[name]
            self._save_schedule()
    
    def list_schedules(self) -> List[Dict]:
        """Список всех задач"""
        return [
            {'name': name, **config}
            for name, config in self.schedule.items()
        ]
    
    def run_scheduled_tests(self):
        """Запуск запланированных тестов"""
        now = datetime.now()
        
        for name, config in self.schedule.items():
            if not config.get('enabled', True):
                continue
            
            # Простая проверка расписания (можно улучшить с помощью cron parser)
            cron = config.get('cron', '')
            if self._should_run(cron, now, config.get('last_run')):
                print(f"Running scheduled test: {name}")
                self._run_test(name, config['command'])
                config['last_run'] = now.isoformat()
                self._save_schedule()
    
    def _should_run(self, cron: str, now: datetime, last_run: Optional[str]) -> bool:
        """Проверка, нужно ли запускать тест"""
        # Простая реализация - можно улучшить
        if not cron:
            return False
        
        # Формат: "HH:MM" или "every N minutes" или "daily"
        if cron.startswith('every '):
            minutes = int(cron.split()[1])
            if last_run:
                last = datetime.fromisoformat(last_run)
                return (now - last).total_seconds() >= minutes * 60
            return True
        
        elif cron == 'daily':
            if last_run:
                last = datetime.fromisoformat(last_run)
                return now.date() > last.date()
            return True
        
        elif ':' in cron:
            # Формат времени HH:MM
            try:
                hour, minute = map(int, cron.split(':'))
                return now.hour == hour and now.minute == minute
            except:
                return False
        
        return False
    
    def _run_test(self, name: str, command: List[str]):
        """Запуск теста"""
        try:
            script_dir = Path(__file__).parent
            full_command = [sys.executable] + [str(script_dir / cmd) if not Path(cmd).is_absolute() else cmd 
                                             for cmd in command]
            
            result = subprocess.run(full_command, capture_output=True, text=True)
            
            print(f"Test {name} completed with exit code: {result.returncode}")
            if result.stdout:
                print(result.stdout)
            if result.stderr:
                print(result.stderr, file=sys.stderr)
            
            return result.returncode == 0
        except Exception as e:
            print(f"Error running test {name}: {e}")
            return False


def main():
    """Главная функция для запуска планировщика"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Chaos Monkey Test Scheduler')
    parser.add_argument('--schedule-file', type=str, default='./chaos_schedule.json',
                       help='Файл с расписанием')
    parser.add_argument('--add', type=str, metavar='NAME',
                       help='Добавить задачу в расписание')
    parser.add_argument('--cron', type=str,
                       help='Cron выражение (например: "HH:MM" или "every N minutes" или "daily")')
    parser.add_argument('--command', type=str, nargs='+',
                       help='Команда для запуска')
    parser.add_argument('--list', action='store_true',
                       help='Список всех задач')
    parser.add_argument('--remove', type=str,
                       help='Удалить задачу')
    parser.add_argument('--run', action='store_true',
                       help='Запустить запланированные тесты')
    parser.add_argument('--daemon', action='store_true',
                       help='Запустить в режиме демона (проверка каждую минуту)')
    
    args = parser.parse_args()
    
    scheduler = TestScheduler(args.schedule_file)
    
    if args.add:
        if not args.cron or not args.command:
            print("Error: --cron and --command required for --add")
            sys.exit(1)
        scheduler.add_schedule(args.add, args.cron, args.command)
        print(f"✅ Задача '{args.add}' добавлена в расписание")
    
    elif args.list:
        schedules = scheduler.list_schedules()
        if schedules:
            print("\nЗапланированные задачи:")
            for sched in schedules:
                print(f"  - {sched['name']}: {sched['cron']} {'(enabled)' if sched.get('enabled') else '(disabled)'}")
        else:
            print("Нет запланированных задач")
    
    elif args.remove:
        scheduler.remove_schedule(args.remove)
        print(f"✅ Задача '{args.remove}' удалена")
    
    elif args.run:
        scheduler.run_scheduled_tests()
    
    elif args.daemon:
        print("Запуск планировщика в режиме демона...")
        print("Нажмите Ctrl+C для остановки")
        try:
            while True:
                scheduler.run_scheduled_tests()
                time.sleep(60)  # Проверка каждую минуту
        except KeyboardInterrupt:
            print("\nПланировщик остановлен")
    
    else:
        parser.print_help()


if __name__ == '__main__':
    main()

