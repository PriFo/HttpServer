#!/bin/bash
# Скрипт для тестирования API клиентов с поддержкой поля country
# Использование: bash test-client-api.sh

BASE_URL="http://127.0.0.1:9999"

echo "=== Тестирование API клиентов с полем country ==="
echo ""

# 1. Создание клиента с полем country
echo "1. Создание клиента с полем country..."
TIMESTAMP=$(date +%Y%m%d%H%M%S)
CREATE_BODY=$(cat <<EOF
{
  "name": "Тестовый клиент $TIMESTAMP",
  "legal_name": "ООО Тестовый клиент",
  "description": "Клиент для тестирования поля country",
  "contact_email": "test@example.com",
  "contact_phone": "+7 (999) 123-45-67",
  "tax_id": "1234567890",
  "country": "RU"
}
EOF
)

CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/clients" \
  -H "Content-Type: application/json" \
  -d "$CREATE_BODY")

if [ $? -eq 0 ]; then
  CLIENT_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":[0-9]*' | grep -o '[0-9]*' | head -1)
  CLIENT_COUNTRY=$(echo "$CREATE_RESPONSE" | grep -o '"country":"[^"]*"' | cut -d'"' -f4)
  echo "✓ Клиент создан успешно. ID: $CLIENT_ID"
  echo "  Country: $CLIENT_COUNTRY"
  echo ""
else
  echo "✗ Ошибка при создании клиента"
  exit 1
fi

# 2. Получение клиента
echo "2. Получение клиента по ID..."
GET_RESPONSE=$(curl -s -X GET "$BASE_URL/api/clients/$CLIENT_ID" \
  -H "Content-Type: application/json")

if [ $? -eq 0 ]; then
  GET_COUNTRY=$(echo "$GET_RESPONSE" | grep -o '"country":"[^"]*"' | cut -d'"' -f4)
  GET_NAME=$(echo "$GET_RESPONSE" | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
  echo "✓ Клиент получен успешно"
  echo "  ID: $CLIENT_ID"
  echo "  Name: $GET_NAME"
  echo "  Country: $GET_COUNTRY"
  echo ""
else
  echo "✗ Ошибка при получении клиента"
  exit 1
fi

# 3. Обновление клиента с изменением country
echo "3. Обновление клиента с изменением country..."
UPDATE_BODY=$(cat <<EOF
{
  "name": "$GET_NAME",
  "legal_name": "ООО Тестовый клиент",
  "description": "Клиент для тестирования поля country",
  "contact_email": "test@example.com",
  "contact_phone": "+7 (999) 123-45-67",
  "tax_id": "1234567890",
  "country": "KZ"
}
EOF
)

UPDATE_RESPONSE=$(curl -s -X PUT "$BASE_URL/api/clients/$CLIENT_ID" \
  -H "Content-Type: application/json" \
  -d "$UPDATE_BODY")

if [ $? -eq 0 ]; then
  UPDATE_COUNTRY=$(echo "$UPDATE_RESPONSE" | grep -o '"country":"[^"]*"' | cut -d'"' -f4)
  echo "✓ Клиент обновлен успешно"
  echo "  Country изменен на: $UPDATE_COUNTRY"
  echo ""
else
  echo "✗ Ошибка при обновлении клиента"
  exit 1
fi

# 4. Проверка, что country сохранился
echo "4. Проверка сохранения country..."
VERIFY_RESPONSE=$(curl -s -X GET "$BASE_URL/api/clients/$CLIENT_ID" \
  -H "Content-Type: application/json")

if [ $? -eq 0 ]; then
  VERIFY_COUNTRY=$(echo "$VERIFY_RESPONSE" | grep -o '"country":"[^"]*"' | cut -d'"' -f4)
  if [ "$VERIFY_COUNTRY" = "KZ" ]; then
    echo "✓ Country успешно сохранен: $VERIFY_COUNTRY"
  else
    echo "✗ Country не сохранен корректно. Ожидалось: KZ, получено: $VERIFY_COUNTRY"
    exit 1
  fi
  echo ""
else
  echo "✗ Ошибка при проверке"
  exit 1
fi

# 5. Получение списка клиентов
echo "5. Получение списка клиентов..."
LIST_RESPONSE=$(curl -s -X GET "$BASE_URL/api/clients" \
  -H "Content-Type: application/json")

if [ $? -eq 0 ]; then
  LIST_COUNTRY=$(echo "$LIST_RESPONSE" | grep -o "\"id\":$CLIENT_ID[^}]*\"country\":\"[^\"]*\"" | grep -o '"country":"[^"]*"' | cut -d'"' -f4)
  if [ -n "$LIST_COUNTRY" ]; then
    echo "✓ Клиент найден в списке"
    echo "  Country в списке: $LIST_COUNTRY"
    if [ "$LIST_COUNTRY" = "KZ" ]; then
      echo "✓ Country корректно отображается в списке"
    else
      echo "✗ Country некорректно в списке. Ожидалось: KZ, получено: $LIST_COUNTRY"
    fi
  else
    echo "✗ Клиент не найден в списке"
  fi
  echo ""
else
  echo "✗ Ошибка при получении списка"
  exit 1
fi

echo "=== Все тесты пройдены успешно! ==="
echo ""
echo "Созданный клиент ID: $CLIENT_ID"
echo "Для удаления используйте: curl -X DELETE $BASE_URL/api/clients/$CLIENT_ID"

