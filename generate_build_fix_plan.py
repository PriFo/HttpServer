#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Скрипт для генерации плана исправления ошибок компиляции Go проекта.
Парсит ошибки компилятора и создает промпты для каждого файла.
"""

import re
import os
import sys
from collections import defaultdict
from pathlib import Path


def parse_go_errors(error_file):
    """
    Парсит ошибки компиляции Go из файла.
    Формат ошибок Go:
    # package_name
    file.go:line:column: error message
    или
    file.go:line:column: error message
        additional context
    """
    errors_by_file = defaultdict(list)
    
    with open(error_file, 'r', encoding='utf-8') as f:
        content = f.read()
    
    lines = content.split('\n')
    
    current_file = None
    current_error = None
    error_context = []
    current_package = None
    
    for line in lines:
        original_line = line
        line = line.strip()
        
        # Пропускаем пустые строки
        if not line:
            continue
        
        # Заголовок пакета: # package_name
        if line.startswith('# '):
            current_package = line[2:].strip()
            continue
            
        # Паттерн для ошибки: file.go:line:column: message
        # Поддерживаем как прямые, так и обратные слеши
        match = re.match(r'^([^:]+\.go):(\d+):(\d+):\s*(.+)$', line)
        if match:
            # Сохраняем предыдущую ошибку
            if current_file and current_error:
                errors_by_file[current_file].append({
                    'line': current_error['line'],
                    'column': current_error['column'],
                    'message': current_error['message'],
                    'context': '\n'.join(error_context) if error_context else None,
                    'package': current_package
                })
            
            # Начинаем новую ошибку
            current_file = match.group(1)
            current_error = {
                'line': int(match.group(2)),
                'column': int(match.group(3)),
                'message': match.group(4)
            }
            error_context = []
        elif (line.startswith('\t') or line.startswith('    ')) and current_error:
            # Дополнительный контекст ошибки (с табуляцией или пробелами)
            error_context.append(line.strip())
        elif line == "too many errors":
            # Специальная строка "too many errors"
            if current_file and current_error:
                errors_by_file[current_file].append({
                    'line': current_error['line'],
                    'column': current_error['column'],
                    'message': current_error['message'] + " (too many errors)",
                    'context': '\n'.join(error_context) if error_context else None,
                    'package': current_package
                })
            current_file = None
            current_error = None
            error_context = []
    
    # Сохраняем последнюю ошибку
    if current_file and current_error:
        errors_by_file[current_file].append({
            'line': current_error['line'],
            'column': current_error['column'],
            'message': current_error['message'],
            'context': '\n'.join(error_context) if error_context else None,
            'package': current_package
        })
    
    return errors_by_file


def normalize_file_path(file_path):
    """Нормализует путь к файлу (заменяет обратные слеши на прямые для Windows)"""
    return file_path.replace('\\', '/')


def explain_error_type(error_msg):
    """
    Объясняет тип ошибки Go компилятора.
    Не указывает, как исправить, только объясняет суть ошибки.
    """
    error_lower = error_msg.lower()
    
    if 'undefined' in error_lower:
        return """
Ошибка "undefined" означает, что компилятор не может найти определение символа (переменной, функции, типа, пакета).

Тонкости этой ошибки:
- Символ может быть не объявлен в текущем пакете или в импортированных пакетах
- Неправильный импорт пакета - возможно, пакет импортирован, но символ в нем не существует или имеет другое имя
- Символ может быть объявлен в другом файле того же пакета, но не экспортирован (начинается с маленькой буквы) - в Go экспортируются только символы с заглавной буквы
- Опечатка в имени символа - очень частая причина
- Файл с определением может не быть включен в компиляцию (не в той директории или не имеет правильного package)
- Тип может быть определен в другом пакете, но не импортирован
- Может быть проблема с циклическими зависимостями между пакетами
"""
    
    if 'imported and not used' in error_lower:
        return """
Ошибка "imported and not used" означает, что пакет импортирован, но ни один его символ не используется в коде.

Тонкости этой ошибки:
- В Go неиспользуемые импорты считаются ошибкой компиляции
- Импорт может быть оставлен после рефакторинга, когда код, использующий пакет, был удален
- Может быть опечатка в имени импортируемого пакета
- Пакет может быть импортирован с алиасом, но алиас не используется
- Может быть импорт для side-effect (например, _ "package"), но забыт подчеркивающий символ
"""
    
    if 'too many arguments' in error_lower:
        return """
Ошибка "too many arguments" означает, что функция вызывается с большим количеством аргументов, чем она принимает.

Тонкости этой ошибки:
- Компилятор показывает "have" - какие аргументы переданы
- Компилятор показывает "want" - какие аргументы ожидаются
- Может быть изменена сигнатура функции, но не обновлены все места вызова
- Может быть передано лишнее значение из-за копирования кода
- Может быть проблема с variadic функциями (с ...)
"""
    
    if 'too few arguments' in error_lower:
        return """
Ошибка "too few arguments" означает, что функции передано недостаточно аргументов.

Тонкости этой ошибки:
- Компилятор показывает, сколько аргументов передано и сколько ожидается
- Может быть удален обязательный параметр из сигнатуры функции
- Может быть забыт аргумент при вызове функции
"""
    
    if 'cannot use' in error_lower:
        return """
Ошибка "cannot use" означает несовместимость типов - переменная одного типа используется там, где ожидается другой тип.

Тонкости этой ошибки:
- Go строго типизированный язык, неявные преобразования типов ограничены
- Компилятор показывает, какой тип передан и какой ожидается
- Может потребоваться явное преобразование типа
- Может быть проблема с интерфейсами - тип не реализует требуемый интерфейс
- Может быть проблема с указателями - передается значение вместо указателя или наоборот
"""
    
    if 'not declared' in error_lower:
        return """
Ошибка "not declared" означает, что переменная или функция используется, но не объявлена в области видимости.

Тонкости этой ошибки:
- Переменная может быть объявлена в другой области видимости (внутри if, for и т.д.)
- Переменная может быть объявлена после использования (в Go порядок объявления важен)
- Может быть опечатка в имени переменной
- Может быть проблема с shadowing - переменная с таким именем объявлена во внешней области, но не видна
"""

    if 'too many errors' in error_lower:
        return """
Сообщение "too many errors" означает, что компилятор обнаружил слишком много ошибок в файле и прекратил дальнейший анализ.

Тонкости этой ошибки:
- Это не отдельная ошибка, а индикатор того, что в файле накопилось слишком много ошибок
- Компилятор останавливается после определенного количества ошибок (обычно 10)
- Нужно исправить первые ошибки, чтобы увидеть остальные
- Часто первые ошибки вызывают каскад последующих (например, неопределенный тип вызывает ошибки везде, где он используется)
- Рекомендуется исправлять ошибки последовательно, начиная с первых
"""

    return """
Общая ошибка компиляции Go. 

Тонкости компиляции Go:
- Go компилятор строгий и требует явного объявления всех символов
- Порядок объявления важен - нельзя использовать символ до его объявления
- Экспорт символов зависит от регистра первой буквы
- Импорты должны использоваться, иначе это ошибка
- Типы должны совпадать точно, неявные преобразования ограничены
- Все переменные должны быть использованы (кроме _)
"""


def generate_prompt(file_path, errors):
    """Генерирует промпт для исправления ошибок в одном файле"""
    
    file_path_normalized = normalize_file_path(file_path)
    
    # Собираем все ошибки
    error_details = []
    for i, err in enumerate(errors, 1):
        detail = f"Ошибка {i}:\n"
        detail += f"  Строка: {err['line']}, Колонка: {err['column']}\n"
        detail += f"  Сообщение компилятора: {err['message']}\n"
        if err.get('package'):
            detail += f"  Пакет: {err['package']}\n"
        if err.get('context'):
            detail += f"  Дополнительный контекст:\n{err['context']}\n"
        error_details.append(detail)
    
    # Определяем типы ошибок для объяснений
    error_types_explanations = []
    seen_types = set()
    for err in errors:
        msg_lower = err['message'].lower()
        explanation = explain_error_type(err['message'])
        # Добавляем объяснение только один раз для каждого типа
        explanation_key = explanation[:50]  # Используем начало как ключ
        if explanation_key not in seen_types:
            seen_types.add(explanation_key)
            error_types_explanations.append(explanation)
    
    prompt = f"""Исправь все ошибки компиляции в файле `{file_path_normalized}`.

## Файл для исправления:
`{file_path_normalized}`

## Ошибки компилятора в этом файле:

{chr(10).join(error_details)}

## Объяснение типов ошибок (тонкости):

{chr(10).join(error_types_explanations)}

## Важные примечания:

- Некоторые зависимые файлы могли создаваться отдельно, т.е. Handler может зависеть от repository, но repository мог не существовать на момент написания handler

- Могут быть простые ошибки в наименовании файлов при импорте

- Но может быть действительно отсутствующий файл - убедись на 150%, что файл действительно отсутствует, прежде чем создавать его

- Проверь все импорты и убедись, что они указывают на существующие файлы с правильными именами

- Проверь после исправления компиляцию файла напрямую и, если есть ошибки - исправь их. Делай циклы "Проверка компиляции - исправление ошибки" пока не будет ошибок. Если после компиляции файла остались ошибки - сообщи об этом.

## Задача:

Проанализируй каждую ошибку компилятора, определи её причину, подумай над решением и исправь все ошибки в файле. 

ВАЖНО: Не исправляй файл построчно - исправь все ошибки в файле за один раз полностью. После исправления проверь компиляцию файла с помощью команды в терминале (PowerShell): `go build ./path/to/file.go` или `go build ./package/path`. Если после исправления остались ошибки - исправь их в цикле до полного устранения всех ошибок компиляции.
"""
    
    return prompt


def main():
    error_file = 'build_errors_full.log'
    
    if not os.path.exists(error_file):
        print(f"Файл {error_file} не найден!")
        return
    
    print(f"Парсинг ошибок из {error_file}...")
    errors_by_file = parse_go_errors(error_file)
    
    if not errors_by_file:
        print("Ошибок компиляции не найдено! Проект собирается без ошибок.")
        return
    
    # Генерируем промпты для каждого файла
    prompts = []
    for file_path, errors in sorted(errors_by_file.items()):
        prompt = generate_prompt(file_path, errors)
        prompts.append({
            'file': normalize_file_path(file_path),
            'errors_count': len(errors),
            'prompt': prompt
        })
    
    # Сохраняем план
    plan_file = 'build_fix_plan.md'
    with open(plan_file, 'w', encoding='utf-8') as f:
        f.write("# План исправления ошибок компиляции\n\n")
        f.write(f"Всего файлов с ошибками: {len(prompts)}\n")
        f.write(f"Всего ошибок: {sum(p['errors_count'] for p in prompts)}\n")
        f.write(f"Всего промптов: {len(prompts)}\n\n")
        f.write("**Примечание:** Шаги можно выполнять параллельно, так как каждый шаг исправляет отдельный файл.\n\n")
        f.write("---\n\n")
        
        for i, item in enumerate(prompts, 1):
            f.write(f"## Шаг {i}: Исправление `{item['file']}`\n\n")
            f.write(f"**Количество ошибок в файле:** {item['errors_count']}\n\n")
            f.write("**Промпт:**\n\n")
            f.write("```\n")
            f.write(item['prompt'])
            f.write("\n```\n\n")
            f.write("---\n\n")
    
    print(f"\nПлан сохранен в {plan_file}")
    print(f"\nСтатистика:")
    print(f"  - Файлов с ошибками: {len(prompts)}")
    print(f"  - Всего ошибок: {sum(p['errors_count'] for p in prompts)}")
    print(f"  - Промптов создано: {len(prompts)}")
    
    # Также сохраняем отдельные промпты для удобства
    prompts_dir = Path('build_fix_prompts')
    prompts_dir.mkdir(exist_ok=True)
    
    for i, item in enumerate(prompts, 1):
        # Создаем безопасное имя файла
        safe_name = Path(item['file']).stem.replace('\\', '_').replace('/', '_')
        prompt_file = prompts_dir / f"step_{i:02d}_{safe_name}.txt"
        with open(prompt_file, 'w', encoding='utf-8') as f:
            f.write(item['prompt'])
    
    print(f"\nОтдельные промпты сохранены в директории {prompts_dir}/")


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nПрервано пользователем.")
        exit(1)
    except Exception as e:
        print(f"\n\nОШИБКА: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        exit(1)

