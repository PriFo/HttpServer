#!/usr/bin/env python3
"""
Update workload statistics in team.json based on current tasks
"""

import json
import os
from typing import Dict

TODO_DB = os.path.join(os.path.dirname(__file__), '..', 'tasks.json')
TEAM_CONFIG = os.path.join(os.path.dirname(__file__), '..', 'team.json')

def load_json(filepath: str) -> dict:
    """Load JSON file"""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            return json.load(f)
    except FileNotFoundError:
        return {}
    except json.JSONDecodeError:
        print(f"Error: Invalid JSON in {filepath}")
        return {}

def save_json(filepath: str, data: dict):
    """Save JSON file"""
    with open(filepath, 'w', encoding='utf-8') as f:
        json.dump(data, f, indent=2, ensure_ascii=False)

def calculate_workload(tasks: list, team_members: dict) -> dict:
    """Calculate workload for each team member"""
    workload = {member: 0 for members in team_members.values() for member in members}
    
    for task in tasks:
        assigned_to = task.get('assignedTo', '')
        status = task.get('status', 'OPEN')
        priority = task.get('priority', 'MEDIUM')
        
        # Находим разработчика в команде
        for team_name, members in team_members.items():
            if assigned_to == team_name:
                # Распределяем нагрузку между членами команды
                if members:
                    # Пока просто добавляем к первому члену команды
                    # Можно улучшить логику распределения
                    if members[0] in workload:
                        if status == 'IN_PROGRESS':
                            workload[members[0]] += 3
                        elif status == 'OPEN':
                            priority_weights = {
                                'CRITICAL': 5,
                                'HIGH': 3,
                                'MEDIUM': 2,
                                'LOW': 1
                            }
                            workload[members[0]] += priority_weights.get(priority, 2)
            elif assigned_to in members:
                # Назначено конкретному разработчику
                if assigned_to in workload:
                    if status == 'IN_PROGRESS':
                        workload[assigned_to] += 3
                    elif status == 'OPEN':
                        priority_weights = {
                            'CRITICAL': 5,
                            'HIGH': 3,
                            'MEDIUM': 2,
                            'LOW': 1
                        }
                        workload[assigned_to] += priority_weights.get(priority, 2)
    
    return workload

def update_workload():
    """Update workload in team.json"""
    tasks_data = load_json(TODO_DB)
    team_config = load_json(TEAM_CONFIG)
    
    if not tasks_data or 'tasks' not in tasks_data:
        print("No tasks found")
        return
    
    if 'team' not in team_config:
        print("Team configuration not found")
        return
    
    tasks = tasks_data['tasks']
    team_members = team_config['team']
    
    workload = calculate_workload(tasks, team_members)
    
    # Обновляем workload в конфиге
    team_config['workload'] = workload
    
    save_json(TEAM_CONFIG, team_config)
    
    print("✓ Workload updated:")
    for member, load in sorted(workload.items(), key=lambda x: x[1], reverse=True):
        if load > 0:
            print(f"  {member}: {load}")

if __name__ == '__main__':
    update_workload()

