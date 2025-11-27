#!/usr/bin/env python3
"""
Генерация отчета по TODO задачам
"""

import json
from datetime import datetime
from pathlib import Path

PROJECT_ROOT = Path(__file__).parent.parent.parent
TODO_DB = PROJECT_ROOT / ".todos" / "tasks.json"
REPORT_FILE = PROJECT_ROOT / "TODO_REPORT.md"


def load_tasks():
    """Загрузка задач"""
    if TODO_DB.exists():
        with open(TODO_DB, 'r', encoding='utf-8') as f:
            return json.load(f)
    return {"tasks": []}


def generate_report():
    """Генерация отчета"""
    data = load_tasks()
    tasks = data.get('tasks', [])
    
    # Статистика
    total_tasks = len(tasks)
    open_tasks = [t for t in tasks if t.get('status') == 'OPEN']
    closed_tasks = [t for t in tasks if t.get('status') in ['RESOLVED', 'TESTING']]
    
    # По приоритетам
    by_priority = {
        'CRITICAL': [t for t in open_tasks if t.get('priority') == 'CRITICAL'],
        'HIGH': [t for t in open_tasks if t.get('priority') == 'HIGH'],
        'MEDIUM': [t for t in open_tasks if t.get('priority') == 'MEDIUM'],
        'LOW': [t for t in open_tasks if t.get('priority') == 'LOW']
    }
    
    # По типам
    by_type = {}
    for task in open_tasks:
        task_type = task.get('type', 'TODO')
        by_type[task_type] = by_type.get(task_type, 0) + 1
    
    # По файлам
    by_file = {}
    for task in open_tasks:
        file_path = task.get('file', 'unknown')
        by_file[file_path] = by_file.get(file_path, 0) + 1
    
    # Генерация Markdown
    report = f"""# 🎯 Automated TODO Report

**Сгенерировано:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}

## 📈 Быстрая статистика

- **Всего задач:** {total_tasks}
- **Открытых задач:** {len(open_tasks)}
- **Закрытых задач:** {len(closed_tasks)}
- **Процент выполнения:** {int((len(closed_tasks) / total_tasks * 100) if total_tasks > 0 else 0)}%

## 🚨 Критические задачи (требуют немедленного внимания)

"""
    
    if by_priority['CRITICAL']:
        for task in by_priority['CRITICAL'][:10]:  # Показываем первые 10
            report += f"""### {task.get('file', 'unknown')}:{task.get('line', 0)}

- **Описание:** {task.get('description', 'N/A')}
- **Тип:** {task.get('type', 'TODO')}
- **Назначено:** {task.get('assignedTo', 'Не назначено')}
- **Создано:** {task.get('createdAt', 'N/A')[:10]}

"""
    else:
        report += "✅ Критических задач не найдено!\n\n"
    
    report += f"""## ⚠️ Важные задачи (HIGH)

"""
    
    if by_priority['HIGH']:
        for task in by_priority['HIGH'][:10]:
            report += f"""- `{task.get('file', 'unknown')}:{task.get('line', 0)}` - {task.get('description', 'N/A')[:60]}...\n"""
    else:
        report += "✅ Важных задач не найдено!\n"
    
    report += f"""
## 📊 Распределение по приоритетам

- **CRITICAL:** {len(by_priority['CRITICAL'])}
- **HIGH:** {len(by_priority['HIGH'])}
- **MEDIUM:** {len(by_priority['MEDIUM'])}
- **LOW:** {len(by_priority['LOW'])}

## 📋 Распределение по типам

"""
    
    for task_type, count in sorted(by_type.items(), key=lambda x: x[1], reverse=True):
        report += f"- **{task_type}:** {count}\n"
    
    report += f"""
## 📁 Топ файлов с наибольшим количеством TODO

"""
    
    top_files = sorted(by_file.items(), key=lambda x: x[1], reverse=True)[:10]
    for file_path, count in top_files:
        report += f"- `{file_path}` - {count} задач\n"
    
    report += f"""
## 🎯 Следующие действия

1. Просмотреть критические задачи
2. Назначить нераспределенные задачи
3. Обновить статус задач в процессе
4. Закрыть выполненные задачи

---
*Отчет автоматически генерируется системой управления TODO*
"""
    
    # Сохранение отчета
    with open(REPORT_FILE, 'w', encoding='utf-8') as f:
        f.write(report)
    
    print(f"✅ Отчет сгенерирован: {REPORT_FILE}")
    print(f"📊 Статистика:")
    print(f"   - Всего задач: {total_tasks}")
    print(f"   - Открытых: {len(open_tasks)}")
    print(f"   - Критических: {len(by_priority['CRITICAL'])}")


if __name__ == "__main__":
    generate_report()

