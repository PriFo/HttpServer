#!/usr/bin/env python3
"""
Система автоматического назначения TODO задач
"""

import json
import os
import sys
from typing import Dict, List, Optional
from collections import defaultdict

class AssignmentEngine:
    def __init__(self, team_config_path: str, tasks_db_path: str):
        self.team_config_path = team_config_path
        self.tasks_db_path = tasks_db_path
        self.team_config = self._load_team_config()
        self.tasks_db = self._load_tasks_db()
        
    def _load_team_config(self) -> Dict:
        """Загружает конфигурацию команды"""
        if not os.path.exists(self.team_config_path):
            return {
                "team": {
                    "backend-team": ["backend-dev-1", "backend-dev-2"],
                    "frontend-team": ["frontend-dev-1", "frontend-dev-2"],
                    "devops": ["devops-dev-1"]
                },
                "specialties": {
                    "go": ["backend-dev-1", "backend-dev-2"],
                    "typescript": ["frontend-dev-1", "frontend-dev-2"],
                    "react": ["frontend-dev-1", "frontend-dev-2"]
                },
                "workload": {}
            }
        
        with open(self.team_config_path, 'r', encoding='utf-8') as f:
            return json.load(f)
    
    def _load_tasks_db(self) -> Dict:
        """Загружает базу данных задач"""
        if not os.path.exists(self.tasks_db_path):
            return {"tasks": []}
        
        with open(self.tasks_db_path, 'r', encoding='utf-8') as f:
            return json.load(f)
    
    def _save_tasks_db(self):
        """Сохраняет базу данных задач"""
        os.makedirs(os.path.dirname(self.tasks_db_path), exist_ok=True)
        with open(self.tasks_db_path, 'w', encoding='utf-8') as f:
            json.dump(self.tasks_db, f, indent=2, ensure_ascii=False)
    
    def _get_file_extension(self, file_path: str) -> str:
        """Определяет расширение файла"""
        return os.path.splitext(file_path)[1].lower()
    
    def _get_technology_from_file(self, file_path: str) -> str:
        """Определяет технологию по файлу"""
        ext = self._get_file_extension(file_path)
        tech_map = {
            '.go': 'go',
            '.ts': 'typescript',
            '.tsx': 'typescript',
            '.js': 'javascript',
            '.jsx': 'javascript',
            '.py': 'python',
            '.sh': 'bash',
            '.ps1': 'powershell'
        }
        return tech_map.get(ext, 'other')
    
    def _get_category_from_file(self, file_path: str) -> str:
        """Определяет категорию по пути файла"""
        if 'frontend' in file_path or 'components' in file_path:
            return 'frontend'
        elif 'backend' in file_path or 'server' in file_path or 'cmd' in file_path:
            return 'backend'
        elif 'scripts' in file_path or 'docker' in file_path:
            return 'devops'
        return 'other'
    
    def _calculate_workload(self) -> Dict[str, int]:
        """Рассчитывает текущую нагрузку разработчиков"""
        workload = defaultdict(int)
        
        for task in self.tasks_db.get("tasks", []):
            if task.get("status") in ["OPEN", "IN_PROGRESS"]:
                assigned = task.get("assignedTo")
                if assigned:
                    # Учитываем приоритет при расчете нагрузки
                    priority_weight = {
                        "CRITICAL": 4,
                        "HIGH": 2,
                        "MEDIUM": 1,
                        "LOW": 0.5
                    }
                    weight = priority_weight.get(task.get("priority", "MEDIUM"), 1)
                    workload[assigned] += weight
        
        return workload
    
    def _find_best_assignee(self, task: Dict) -> Optional[str]:
        """Находит лучшего исполнителя для задачи"""
        category = task.get("category", "other")
        file_path = task.get("file", "")
        technology = self._get_technology_from_file(file_path)
        priority = task.get("priority", "MEDIUM")
        
        # Получаем специалистов по технологии
        specialties = self.team_config.get("specialties", {})
        candidates = specialties.get(technology, [])
        
        # Если нет специалистов по технологии, используем категорию
        if not candidates:
            team_map = {
                "frontend": "frontend-team",
                "backend": "backend-team",
                "devops": "devops"
            }
            team_name = team_map.get(category, "backend-team")
            team = self.team_config.get("team", {}).get(team_name, [])
            candidates = team
        
        if not candidates:
            return None
        
        # Рассчитываем нагрузку
        workload = self._calculate_workload()
        
        # Выбираем кандидата с минимальной нагрузкой
        best_candidate = None
        min_workload = float('inf')
        
        for candidate in candidates:
            candidate_workload = workload.get(candidate, 0)
            if candidate_workload < min_workload:
                min_workload = candidate_workload
                best_candidate = candidate
        
        return best_candidate
    
    def assign_unassigned_tasks(self) -> int:
        """Назначает не назначенные задачи"""
        assigned_count = 0
        
        for task in self.tasks_db.get("tasks", []):
            if not task.get("assignedTo") and task.get("status") == "OPEN":
                assignee = self._find_best_assignee(task)
                if assignee:
                    task["assignedTo"] = assignee
                    assigned_count += 1
                    print(f"✅ Назначено: {task['id']} -> {assignee}")
        
        if assigned_count > 0:
            self._save_tasks_db()
        
        return assigned_count
    
    def reassign_by_priority(self):
        """Перераспределяет задачи по приоритету"""
        # Сначала назначаем CRITICAL задачи
        for task in self.tasks_db.get("tasks", []):
            if task.get("priority") == "CRITICAL" and not task.get("assignedTo"):
                assignee = self._find_best_assignee(task)
                if assignee:
                    task["assignedTo"] = assignee
        
        self._save_tasks_db()

def main():
    project_dir = os.environ.get("PROJECT_DIR", ".")
    team_config = os.path.join(project_dir, ".todos", "team.json")
    tasks_db = os.path.join(project_dir, ".todos", "tasks.json")
    
    if len(sys.argv) > 1:
        command = sys.argv[1]
    else:
        command = "assign"
    
    engine = AssignmentEngine(team_config, tasks_db)
    
    if command == "assign":
        count = engine.assign_unassigned_tasks()
        print(f"\n📝 Назначено задач: {count}")
    elif command == "reassign":
        engine.reassign_by_priority()
        print("✅ Задачи перераспределены")
    else:
        print("Использование: assign-tasks.py [assign|reassign]")

if __name__ == "__main__":
    main()
