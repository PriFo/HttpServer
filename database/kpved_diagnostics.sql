-- SQL диагностика для выявления проблем с иерархией КПВЭД классификатора
-- Этот файл содержит запросы для диагностики и исправления "осиротевших" узлов

-- ============================================================================
-- ДИАГНОСТИКА: Найти все узлы с несуществующими родителями
-- ============================================================================
-- Этот запрос находит все узлы, у которых указанный parent_code отсутствует в таблице
SELECT 
    child.code,
    child.name,
    child.parent_code AS requested_parent,
    child.level
FROM kpved_classifier child
LEFT JOIN kpved_classifier parent ON child.parent_code = parent.code
WHERE child.parent_code IS NOT NULL 
  AND child.parent_code != ''
  AND parent.code IS NULL
ORDER BY child.code;

-- ============================================================================
-- СТАТИСТИКА: Подсчет проблемных узлов
-- ============================================================================
SELECT 
    COUNT(*) AS orphan_nodes_count,
    COUNT(DISTINCT child.parent_code) AS missing_parents_count
FROM kpved_classifier child
LEFT JOIN kpved_classifier parent ON child.parent_code = parent.code
WHERE child.parent_code IS NOT NULL 
  AND child.parent_code != ''
  AND parent.code IS NULL;

-- ============================================================================
-- СПИСОК: Все отсутствующие родительские коды
-- ============================================================================
SELECT DISTINCT child.parent_code AS missing_parent_code
FROM kpved_classifier child
LEFT JOIN kpved_classifier parent ON child.parent_code = parent.code
WHERE child.parent_code IS NOT NULL 
  AND child.parent_code != ''
  AND parent.code IS NULL
ORDER BY child.parent_code;

-- ============================================================================
-- РЕШЕНИЕ 1: Обновить parent_code на ближайшего существующего родителя
-- ============================================================================
-- ВНИМАНИЕ: Этот запрос требует ручной проверки перед выполнением!
-- Он пытается найти ближайшего существующего родителя для каждого проблемного узла
-- 
-- Логика: для кода вида "30.92.1" ищет родителя "30.92", если не найден - "30", если не найден - NULL
UPDATE kpved_classifier
SET parent_code = (
    -- Пытаемся найти родителя, удаляя последний сегмент
    SELECT COALESCE(
        -- Пробуем найти родителя с удалением последнего сегмента после точки
        (SELECT code FROM kpved_classifier 
         WHERE code = substr(kpved_classifier.parent_code, 1, length(kpved_classifier.parent_code) - 
             CASE 
                 WHEN substr(kpved_classifier.parent_code, -2, 1) = '.' 
                 THEN 2  -- Удаляем ".X"
                 WHEN substr(kpved_classifier.parent_code, -3, 1) = '.' 
                 THEN 3  -- Удаляем ".XX"
                 ELSE 0
             END)
         LIMIT 1),
        -- Если не найден, пробуем найти родителя на уровень выше
        (SELECT code FROM kpved_classifier 
         WHERE code = substr(kpved_classifier.parent_code, 1, 
             length(kpved_classifier.parent_code) - 
             length(substr(kpved_classifier.parent_code, instr(kpved_classifier.parent_code || '.', '.', -1) + 1)) - 1)
         LIMIT 1),
        NULL  -- Если ничего не найдено, устанавливаем NULL
    )
)
WHERE code IN (
    SELECT child.code
    FROM kpved_classifier child
    LEFT JOIN kpved_classifier parent ON child.parent_code = parent.code
    WHERE child.parent_code IS NOT NULL 
      AND child.parent_code != ''
      AND parent.code IS NULL
);

-- ============================================================================
-- РЕШЕНИЕ 2: Установить parent_code в NULL для узлов без существующего родителя
-- ============================================================================
-- ВНИМАНИЕ: Это более безопасный вариант, но может нарушить иерархию
-- Используйте только если уверены, что эти узлы должны быть корневыми
UPDATE kpved_classifier
SET parent_code = NULL
WHERE code IN (
    SELECT child.code
    FROM kpved_classifier child
    LEFT JOIN kpved_classifier parent ON child.parent_code = parent.code
    WHERE child.parent_code IS NOT NULL 
      AND child.parent_code != ''
      AND parent.code IS NULL
);

-- ============================================================================
-- РЕШЕНИЕ 3: Создать недостающие родительские узлы (требует ручного заполнения)
-- ============================================================================
-- ВНИМАНИЕ: Этот запрос только показывает, какие узлы нужно создать
-- Вы должны вручную определить правильные названия для этих узлов
SELECT DISTINCT 
    child.parent_code AS missing_code,
    -- Предполагаем уровень на основе формата кода
    CASE 
        WHEN length(child.parent_code) = 1 THEN 1  -- Секция (A-Z)
        WHEN length(child.parent_code) = 2 THEN 2  -- Класс (01-99)
        ELSE (length(replace(child.parent_code, '.', '')) - 1) / 2 + 1  -- Подкласс и ниже
    END AS suggested_level
FROM kpved_classifier child
LEFT JOIN kpved_classifier parent ON child.parent_code = parent.code
WHERE child.parent_code IS NOT NULL 
  AND child.parent_code != ''
  AND parent.code IS NULL
ORDER BY child.parent_code;

-- Пример INSERT для создания недостающего узла (требует ручного заполнения name):
-- INSERT INTO kpved_classifier (code, name, parent_code, level)
-- VALUES ('30.92', 'Название должно быть заполнено вручную', '30', 3);

-- ============================================================================
-- ВАЛИДАЦИЯ: Проверка целостности после исправлений
-- ============================================================================
-- После применения исправлений выполните этот запрос для проверки
SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN 'OK: Все узлы имеют существующих родителей'
        ELSE 'ERROR: Найдено ' || COUNT(*) || ' узлов с несуществующими родителями'
    END AS validation_result
FROM kpved_classifier child
LEFT JOIN kpved_classifier parent ON child.parent_code = parent.code
WHERE child.parent_code IS NOT NULL 
  AND child.parent_code != ''
  AND parent.code IS NULL;

