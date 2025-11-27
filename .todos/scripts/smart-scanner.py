#!/usr/bin/env python3
"""
Smart TODO Scanner with intelligent parsing
Автоматическое сканирование с интеллектуальным парсингом
"""

import re
import json
import os
import hashlib
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass, asdict
from enum import Enum

class Priority(Enum):
    CRITICAL = "CRITICAL"
    HIGH = "HIGH"
    MEDIUM = "MEDIUM"
    LOW = "LOW"

class TaskType(Enum):
    TODO = "TODO"
    FIXME = "FIXME"
    HACK = "HACK"
    REFACTOR = "REFACTOR"

class FileType(Enum):
    BACKEND = "backend"
    FRONTEND = "frontend"
    SCRIPT = "script"
    OTHER = "other"

@dataclass
class TodoTask:
    id: str
    file: str
    line: int
    description: str
    type: str
    priority: str
    status: str
    assignedTo: str
    createdAt: str
    updatedAt: str
    fileType: str
    estimatedHours: int = 2
    actualHours: Optional[int] = None
    dependencies: List[str] = None
    relatedFiles: List[str] = None
    
    def __post_init__(self):
        if self.dependencies is None:
            self.dependencies = []
        if self.relatedFiles is None:
            self.relatedFiles = []

class SmartTodoScanner:
    """Интеллектуальный сканер TODO с категоризацией"""
    
    def __init__(self, project_dir: str = ".", config_file: str = ".todos/config.json"):
        self.project_dir = Path(project_dir)
        self.config_file = Path(config_file)
        self.tasks_db = Path(".todos/tasks.json")
        self.team_config = Path(".todos/team.json")
        
        # Паттерны для поиска
        self.patterns = {
            'CRITICAL': [
                r'TODO\s*\(\s*CRITICAL\s*\)',
                r'FIXME',
                r'HACK',
                r'panic\(',
                r'not\s+implemented',
                r'BUG:',
                r'CRITICAL:'
            ],
            'HIGH': [
                r'TODO\s*\(\s*HIGH\s*\)',
                r'URGENT',
                r'IMPORTANT',
                r'FIX\s+THIS'
            ],
            'MEDIUM': [
                r'TODO\s*\(\s*MEDIUM\s*\)',
                r'OPTIMIZE',
                r'REFACTOR',
                r'IMPROVE'
            ],
            'LOW': [
                r'TODO\s*\(\s*LOW\s*\)',
                r'CLEANUP',
                r'DOCUMENT',
                r'ENHANCE'
            ]
        }
        
        # Расширения файлов
        self.file_extensions = {
            FileType.BACKEND: ['.go', '.py', '.java', '.rb', '.php', '.rs'],
            FileType.FRONTEND: ['.ts', '.tsx', '.js', '.jsx', '.vue', '.svelte'],
            FileType.SCRIPT: ['.sh', '.ps1', '.bat', '.zsh', '.fish']
        }
        
        # Игнорируемые паттерны
        self.ignore_patterns = [
            'node_modules',
            '.git',
            'vendor',
            'dist',
            'build',
            '__pycache__',
            '.venv',
            'venv',
            '*.min.js',
            '*.min.css'
        ]
    
    def load_config(self) -> Dict:
        """Загрузка конфигурации"""
        if self.config_file.exists():
            with open(self.config_file, 'r', encoding='utf-8') as f:
                return json.load(f)
        return {}
    
    def load_tasks(self) -> List[Dict]:
        """Загрузка существующих задач"""
        if self.tasks_db.exists():
            with open(self.tasks_db, 'r', encoding='utf-8') as f:
                data = json.load(f)
                return data.get('tasks', [])
        return []
    
    def save_tasks(self, tasks: List[Dict]):
        """Сохранение задач"""
        self.tasks_db.parent.mkdir(parents=True, exist_ok=True)
        data = {
            'tasks': tasks,
            'version': '1.0.0',
            'lastScan': datetime.utcnow().isoformat() + 'Z'
        }
        with open(self.tasks_db, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
    
    def generate_id(self, file: str, line: int) -> str:
        """Генерация уникального ID задачи"""
        content = f"{file}:{line}"
        return hashlib.sha256(content.encode()).hexdigest()[:8]
    
    def detect_file_type(self, file_path: Path) -> FileType:
        """Определение типа файла"""
        ext = file_path.suffix.lower()
        
        for file_type, extensions in self.file_extensions.items():
            if ext in extensions:
                return file_type
        return FileType.OTHER
    
    def should_ignore(self, file_path: Path) -> bool:
        """Проверка, нужно ли игнорировать файл"""
        path_str = str(file_path)
        for pattern in self.ignore_patterns:
            if pattern in path_str:
                return True
        return False
    
    def classify_priority(self, line: str) -> Priority:
        """Классификация приоритета"""
        line_upper = line.upper()
        
        # Проверка критических паттернов
        for pattern in self.patterns['CRITICAL']:
            if re.search(pattern, line_upper, re.IGNORECASE):
                return Priority.CRITICAL
        
        # Проверка высокого приоритета
        for pattern in self.patterns['HIGH']:
            if re.search(pattern, line_upper, re.IGNORECASE):
                return Priority.HIGH
        
        # Проверка среднего приоритета
        for pattern in self.patterns['MEDIUM']:
            if re.search(pattern, line_upper, re.IGNORECASE):
                return Priority.MEDIUM
        
        # Проверка низкого приоритета
        for pattern in self.patterns['LOW']:
            if re.search(pattern, line_upper, re.IGNORECASE):
                return Priority.LOW
        
        # По умолчанию
        if 'FIXME' in line_upper or 'HACK' in line_upper:
            return Priority.HIGH
        return Priority.MEDIUM
    
    def classify_type(self, line: str) -> TaskType:
        """Классификация типа задачи"""
        line_upper = line.upper()
        
        if 'FIXME' in line_upper:
            return TaskType.FIXME
        elif 'HACK' in line_upper:
            return TaskType.HACK
        elif 'REFACTOR' in line_upper or 'OPTIMIZE' in line_upper:
            return TaskType.REFACTOR
        else:
            return TaskType.TODO
    
    def extract_description(self, line: str) -> str:
        """Извлечение описания из строки"""
        # Удаляем комментарии
        for comment_prefix in ['//', '#', '/*', '*/']:
            if comment_prefix in line:
                parts = line.split(comment_prefix, 1)
                if len(parts) > 1:
                    line = parts[1].strip()
        
        # Удаляем маркеры TODO/FIXME/HACK
        patterns = [
            r'^(TODO|FIXME|HACK)\s*\([^)]*\)\s*:?\s*',
            r'^(TODO|FIXME|HACK)\s*:?\s*',
        ]
        
        for pattern in patterns:
            line = re.sub(pattern, '', line, flags=re.IGNORECASE)
        
        line = line.strip()
        
        if not line or len(line) < 3:
            return "No description"
        
        return line[:200]  # Ограничиваем длину
    
    def auto_assign(self, file_type: FileType, priority: Priority) -> str:
        """Автоматическое назначение задачи"""
        if not self.team_config.exists():
            return "unassigned"
        
        try:
            with open(self.team_config, 'r', encoding='utf-8') as f:
                team_data = json.load(f)
            
            if file_type == FileType.BACKEND:
                return "backend-team"
            elif file_type == FileType.FRONTEND:
                return "frontend-team"
            elif file_type == FileType.SCRIPT:
                return "devops"
            else:
                return "unassigned"
        except:
            return "unassigned"
    
    def scan_file(self, file_path: Path) -> List[TodoTask]:
        """Сканирование одного файла"""
        tasks = []
        file_type = self.detect_file_type(file_path)
        
        try:
            with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                for line_num, line in enumerate(f, 1):
                    # Поиск TODO/FIXME/HACK
                    if re.search(r'TODO|FIXME|HACK', line, re.IGNORECASE):
                        priority = self.classify_priority(line)
                        task_type = self.classify_type(line)
                        description = self.extract_description(line)
                        
                        task_id = self.generate_id(str(file_path), line_num)
                        assigned_to = self.auto_assign(file_type, priority)
                        now = datetime.utcnow().isoformat() + 'Z'
                        
                        task = TodoTask(
                            id=task_id,
                            file=str(file_path.relative_to(self.project_dir)),
                            line=line_num,
                            description=description,
                            type=task_type.value,
                            priority=priority.value,
                            status="OPEN",
                            assignedTo=assigned_to,
                            createdAt=now,
                            updatedAt=now,
                            fileType=file_type.value
                        )
                        
                        tasks.append(task)
        except Exception as e:
            print(f"Ошибка при сканировании {file_path}: {e}")
        
        return tasks
    
    def scan_project(self) -> Tuple[int, int]:
        """Сканирование всего проекта"""
        existing_tasks = {task['id']: task for task in self.load_tasks()}
        new_tasks = []
        updated_count = 0
        
        # Сканирование всех файлов
        for file_path in self.project_dir.rglob('*'):
            if not file_path.is_file():
                continue
            
            if self.should_ignore(file_path):
                continue
            
            # Проверяем расширение файла
            if file_path.suffix.lower() in [ext for exts in self.file_extensions.values() for ext in exts]:
                tasks = self.scan_file(file_path)
                
                for task in tasks:
                    task_dict = asdict(task)
                    
                    if task.id in existing_tasks:
                        # Обновляем существующую задачу если она OPEN
                        existing = existing_tasks[task.id]
                        if existing.get('status') == 'OPEN':
                            existing['updatedAt'] = task_dict['updatedAt']
                            updated_count += 1
                    else:
                        # Новая задача
                        new_tasks.append(task_dict)
        
        # Объединяем задачи
        all_tasks = list(existing_tasks.values()) + new_tasks
        
        # Сохраняем
        self.save_tasks(all_tasks)
        
        return len(new_tasks), updated_count
    
    def run(self):
        """Запуск сканирования"""
        print("🔄 Начало интеллектуального сканирования TODO...")
        
        new_count, updated_count = self.scan_project()
        
        print(f"✅ Сканирование завершено!")
        print(f"📊 Создано новых задач: {new_count}")
        print(f"📊 Обновлено задач: {updated_count}")
        
        # Статистика
        tasks = self.load_tasks()
        critical = len([t for t in tasks if t.get('priority') == 'CRITICAL' and t.get('status') == 'OPEN'])
        high = len([t for t in tasks if t.get('priority') == 'HIGH' and t.get('status') == 'OPEN'])
        
        if critical > 0:
            print(f"⚠️  Критических задач: {critical}")
        if high > 0:
            print(f"⚠️  Задач с высоким приоритетом: {high}")

if __name__ == "__main__":
    import sys
    
    project_dir = sys.argv[1] if len(sys.argv) > 1 else "."
    scanner = SmartTodoScanner(project_dir)
    scanner.run()

