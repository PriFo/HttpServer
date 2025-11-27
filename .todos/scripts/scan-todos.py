#!/usr/bin/env python3
"""
Автоматизированная система сканирования TODO, FIXME, HACK
"""

import os
import re
import json
import hashlib
from datetime import datetime
from pathlib import Path
from typing import List, Dict, Optional, Tuple

# Конфигурация
PROJECT_ROOT = Path(__file__).parent.parent.parent
TODO_DB = PROJECT_ROOT / ".todos" / "tasks.json"
TEAM_CONFIG = PROJECT_ROOT / ".todos" / "team.json"

# Паттерны для поиска TODO
PATTERNS = {
    'CRITICAL': [
        r'TODO\s*\(\s*CRITICAL\s*\):',
        r'FIXME\s*\(\s*CRITICAL\s*\):',
        r'panic\(',
        r'not\s+implemented',
        r'XXX\s*CRITICAL'
    ],
    'HIGH': [
        r'TODO\s*\(\s*HIGH\s*\):',
        r'FIXME\s*\(\s*HIGH\s*\):',
        r'FIXME',
        r'HACK',
        r'XXX',
        r'BUG:',
        r'not\s+implemented\s+yet'
    ],
    'MEDIUM': [
        r'TODO\s*\(\s*MEDIUM\s*\):',
        r'TODO\s*\(\s*OPTIMIZE\s*\):',
        r'TODO\s*\(\s*REFACTOR\s*\):',
        r'optimize',
        r'refactor',
        r'improve'
    ],
    'LOW': [
        r'TODO\s*\(\s*LOW\s*\):',
        r'TODO\s*\(\s*CLEANUP\s*\):',
        r'TODO\s*\(\s*DOCUMENT\s*\):',
        r'cleanup',
        r'document',
        r'note:'
    ]
}

# Расширения файлов для сканирования
SCAN_EXTENSIONS = {
    'backend': ['.go'],
    'frontend': ['.ts', '.tsx', '.js', '.jsx'],
    'python': ['.py'],
    'config': ['.json', '.yaml', '.yml', '.toml'],
    'docs': ['.md']
}

# Игнорируемые директории
IGNORE_DIRS = {
    'node_modules', '.next', '.git', 'vendor', 
    'dist', 'build', '.todos', '__pycache__',
    '.venv', 'venv', 'env', '.env'
}


class TodoScanner:
    def __init__(self):
        self.tasks_db = self.load_tasks()
        self.team_config = self.load_team_config()
        
    def load_tasks(self) -> Dict:
        """Загрузка базы данных задач"""
        if TODO_DB.exists():
            with open(TODO_DB, 'r', encoding='utf-8') as f:
                return json.load(f)
        return {"tasks": [], "metadata": {"lastScan": None, "totalTasks": 0, "version": "1.0.0"}}
    
    def load_team_config(self) -> Dict:
        """Загрузка конфигурации команды"""
        if TEAM_CONFIG.exists():
            with open(TEAM_CONFIG, 'r', encoding='utf-8') as f:
                return json.load(f)
        return {"team": {}, "specialties": {}, "workload": {}}
    
    def save_tasks(self):
        """Сохранение базы данных задач"""
        self.tasks_db["metadata"]["lastScan"] = datetime.now().isoformat()
        self.tasks_db["metadata"]["totalTasks"] = len(self.tasks_db["tasks"])
        
        with open(TODO_DB, 'w', encoding='utf-8') as f:
            json.dump(self.tasks_db, f, indent=2, ensure_ascii=False)
    
    def should_ignore(self, path: Path) -> bool:
        """Проверка, нужно ли игнорировать путь"""
        parts = path.parts
        return any(ignore in parts for ignore in IGNORE_DIRS)
    
    def classify_priority(self, line: str) -> Tuple[str, str]:
        """Классификация приоритета и типа TODO"""
        line_upper = line.upper()
        
        # Проверка по паттернам
        for priority in ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW']:
            for pattern in PATTERNS[priority]:
                if re.search(pattern, line, re.IGNORECASE):
                    todo_type = self.detect_type(line)
                    return priority, todo_type
        
        # По умолчанию
        if 'FIXME' in line_upper or 'HACK' in line_upper or 'BUG' in line_upper:
            return 'HIGH', 'FIXME'
        elif 'TODO' in line_upper:
            return 'MEDIUM', 'TODO'
        else:
            return 'LOW', 'TODO'
    
    def detect_type(self, line: str) -> str:
        """Определение типа задачи"""
        line_upper = line.upper()
        if 'FIXME' in line_upper:
            return 'FIXME'
        elif 'HACK' in line_upper:
            return 'HACK'
        elif 'REFACTOR' in line_upper or 'OPTIMIZE' in line_upper:
            return 'REFACTOR'
        else:
            return 'TODO'
    
    def extract_description(self, line: str) -> str:
        """Извлечение описания из строки TODO"""
        # Удаляем комментарии и маркеры
        description = re.sub(r'^\s*//\s*', '', line)
        description = re.sub(r'^\s*#\s*', '', description)
        description = re.sub(r'^\s*\*\s*', '', description)
        
        # Удаляем маркеры TODO/FIXME/HACK
        description = re.sub(r'(TODO|FIXME|HACK|XXX|BUG)\s*\([^)]*\)\s*:?\s*', '', description, flags=re.IGNORECASE)
        description = re.sub(r'(TODO|FIXME|HACK|XXX|BUG)\s*:?\s*', '', description, flags=re.IGNORECASE)
        
        return description.strip()
    
    def detect_file_type(self, file_path: Path) -> str:
        """Определение типа файла (backend/frontend)"""
        ext = file_path.suffix.lower()
        
        if ext in SCAN_EXTENSIONS['backend']:
            return 'backend'
        elif ext in SCAN_EXTENSIONS['frontend']:
            return 'frontend'
        elif ext in SCAN_EXTENSIONS['python']:
            return 'python'
        else:
            return 'other'
    
    def auto_assign(self, file_type: str, priority: str, file_path: Path) -> Optional[str]:
        """Автоматическое назначение задачи"""
        specialties = self.team_config.get('specialties', {})
        workload = self.team_config.get('workload', {})
        
        # Определяем команду на основе типа файла
        team_key = None
        if file_type == 'backend':
            team_key = 'backend'
        elif file_type == 'frontend':
            team_key = 'frontend'
        
        if not team_key or team_key not in specialties:
            return None
        
        # Выбираем разработчика с наименьшей загрузкой
        candidates = specialties.get(team_key, [])
        if not candidates:
            return None
        
        # Находим разработчика с минимальной загрузкой
        min_workload = float('inf')
        assigned = None
        
        for team in candidates:
            for dev in self.team_config.get('team', {}).get(team, []):
                dev_workload = workload.get(dev, 0)
                if dev_workload < min_workload:
                    min_workload = dev_workload
                    assigned = dev
        
        return assigned
    
    def generate_task_id(self, file_path: Path, line_num: int, line: str) -> str:
        """Генерация уникального ID задачи"""
        content = f"{file_path}:{line_num}:{line}"
        return hashlib.md5(content.encode()).hexdigest()[:12]
    
    def task_exists(self, task_id: str) -> bool:
        """Проверка существования задачи"""
        return any(task.get('id') == task_id for task in self.tasks_db['tasks'])
    
    def scan_file(self, file_path: Path) -> List[Dict]:
        """Сканирование одного файла"""
        tasks = []
        
        try:
            with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                lines = f.readlines()
        except Exception as e:
            print(f"⚠️  Ошибка чтения файла {file_path}: {e}")
            return tasks
        
        file_type = self.detect_file_type(file_path)
        relative_path = str(file_path.relative_to(PROJECT_ROOT))
        
        for line_num, line in enumerate(lines, 1):
            # Поиск TODO, FIXME, HACK
            if re.search(r'TODO|FIXME|HACK|XXX|BUG', line, re.IGNORECASE):
                priority, todo_type = self.classify_priority(line)
                description = self.extract_description(line)
                
                if not description:
                    continue
                
                task_id = self.generate_task_id(file_path, line_num, line)
                
                # Проверяем, не существует ли уже задача
                if not self.task_exists(task_id):
                    assigned_to = self.auto_assign(file_type, priority, file_path)
                    
                    task = {
                        "id": task_id,
                        "file": relative_path,
                        "line": line_num,
                        "type": todo_type,
                        "priority": priority,
                        "description": description,
                        "status": "OPEN",
                        "assignedTo": assigned_to,
                        "fileType": file_type,
                        "createdAt": datetime.now().isoformat(),
                        "updatedAt": datetime.now().isoformat(),
                        "estimatedHours": self.estimate_hours(priority),
                        "dependencies": [],
                        "relatedFiles": []
                    }
                    
                    tasks.append(task)
                    print(f"📝 Найдена задача: {relative_path}:{line_num} [{priority}] {description[:50]}")
        
        return tasks
    
    def estimate_hours(self, priority: str) -> float:
        """Оценка времени выполнения"""
        estimates = {
            'CRITICAL': 4.0,
            'HIGH': 2.0,
            'MEDIUM': 1.0,
            'LOW': 0.5
        }
        return estimates.get(priority, 1.0)
    
    def scan_directory(self, directory: Path = None) -> int:
        """Рекурсивное сканирование директории"""
        if directory is None:
            directory = PROJECT_ROOT
        
        total_found = 0
        
        print(f"🔄 Начало сканирования: {directory}")
        
        # Сканирование бэкенда
        for ext in SCAN_EXTENSIONS['backend']:
            for file_path in directory.rglob(f"*{ext}"):
                if not self.should_ignore(file_path):
                    tasks = self.scan_file(file_path)
                    self.tasks_db['tasks'].extend(tasks)
                    total_found += len(tasks)
        
        # Сканирование фронтенда
        for ext in SCAN_EXTENSIONS['frontend']:
            for file_path in directory.rglob(f"*{ext}"):
                if not self.should_ignore(file_path):
                    tasks = self.scan_file(file_path)
                    self.tasks_db['tasks'].extend(tasks)
                    total_found += len(tasks)
        
        # Сканирование Python файлов
        for ext in SCAN_EXTENSIONS['python']:
            for file_path in directory.rglob(f"*{ext}"):
                if not self.should_ignore(file_path):
                    tasks = self.scan_file(file_path)
                    self.tasks_db['tasks'].extend(tasks)
                    total_found += len(tasks)
        
        return total_found
    
    def run(self):
        """Запуск сканирования"""
        print("🚀 Запуск автоматизированного сканирования TODO...")
        print(f"📁 Проект: {PROJECT_ROOT}")
        
        total = self.scan_directory()
        
        self.save_tasks()
        
        print(f"\n✅ Сканирование завершено!")
        print(f"📊 Найдено новых задач: {total}")
        print(f"📋 Всего задач в базе: {len(self.tasks_db['tasks'])}")
        
        # Статистика по приоритетам
        stats = {}
        for task in self.tasks_db['tasks']:
            if task['status'] == 'OPEN':
                priority = task['priority']
                stats[priority] = stats.get(priority, 0) + 1
        
        print("\n📈 Статистика открытых задач:")
        for priority in ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW']:
            count = stats.get(priority, 0)
            if count > 0:
                print(f"  {priority}: {count}")


if __name__ == "__main__":
    scanner = TodoScanner()
    scanner.run()

