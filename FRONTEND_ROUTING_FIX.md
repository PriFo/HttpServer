# Исправление роутинга на странице проекта

## Дата: 2025-11-26

## Проблема

Страница проекта (`/clients/[clientId]/projects/[projectId]`) имела нелогичный роутинг:
1. Breadcrumbs пропускали уровень клиента (`Клиенты -> Проекты -> Проект`)
2. Кнопка "Назад" вела на страницу списка проектов вместо дашборда клиента
3. API возвращал плоскую структуру, а фронтенд ожидал вложенную

## Решение

### Backend (`server/handlers/clients.go`)

Обновлен метод `GetClientProject` для возврата расширенной структуры:

```go
{
  "project": { ...project_fields... },
  "client_name": "Client Name",
  "statistics": {
    "total_benchmarks": 123,
    "approved_benchmarks": 45,
    "avg_quality_score": 0.85
  }
}
```

**Изменения:**
- Получение информации о клиенте для `client_name`
- Подсчет статистики из таблицы `client_benchmarks`
- Формирование структурированного ответа

### Frontend (`frontend/app/clients/[clientId]/projects/[projectId]/page.tsx`)

**1. Обновлен интерфейс `ProjectDetail`:**
```typescript
interface ProjectDetail {
  project: { ... },
  client_name?: string,  // Добавлено
  statistics: { ... },
  benchmarks: [ ... ]
}
```

**2. Исправлены breadcrumbs:**
```typescript
const breadcrumbItems = [
  { label: 'Клиенты', href: '/clients', icon: Building2 },
  { label: project.client_name || 'Клиент', href: `/clients/${clientId}`, icon: Building2 },
  { label: projectInfo.name, href: `#`, icon: Target },
]
```

**3. Исправлена кнопка "Назад":**
```typescript
onClick={() => router.push(`/clients/${clientId}`)}  // Было: `/clients/${clientId}/projects`
```

## Результат

✅ API возвращает структурированные данные с `client_name` и `statistics`  
✅ Breadcrumbs показывают полную иерархию: `Клиенты -> Клиент -> Проект`  
✅ Кнопка "Назад" ведет на дашборд клиента  
✅ Навигация логична и соответствует иерархии данных  

## Дополнительные исправления

- Удален дублирующий файл `server/handlers/databases.go` (используется `databases_gin.go`)
- Исправлена ошибка компиляции с `types.ClientDocument`

## Тестирование

Для проверки работы:
1. Откройте страницу проекта: `/clients/{clientId}/projects/{projectId}`
2. Проверьте breadcrumbs - должен быть уровень клиента
3. Нажмите "Назад" - должен перейти на дашборд клиента
4. Проверьте статистику - должна отображаться корректно

