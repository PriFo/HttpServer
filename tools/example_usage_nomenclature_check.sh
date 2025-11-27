#!/bin/bash
# Пример использования утилиты check_nomenclature_count

# Компиляция утилиты
echo "Компиляция утилиты..."
go build -o check_nomenclature_count.exe check_nomenclature_count.go

if [ $? -ne 0 ]; then
    echo "Ошибка компиляции!"
    exit 1
fi

echo "Утилита скомпилирована успешно!"
echo ""

# Пример 1: Проверка всех номенклатур клиента
echo "Пример 1: Проверка всех номенклатур клиента"
echo "./check_nomenclature_count.exe -db \"путь/к/базе.db\" -client 1"
echo ""

# Пример 2: Проверка номенклатур конкретного проекта
echo "Пример 2: Проверка номенклатур конкретного проекта"
echo "./check_nomenclature_count.exe -db \"путь/к/базе.db\" -client 1 -project 1"
echo ""

# Пример 3: Проверка с детальной информацией
echo "Пример 3: Проверка с детальной информацией"
echo "./check_nomenclature_count.exe -db \"путь/к/базе.db\" -client 1 -details"
echo ""

echo "Готово!"

