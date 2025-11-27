# Сводка изменений: Удаление требования выбора базы данных для КПВЭД

## Проблема
Классификатор КПВЭД не отображался, хотя данные были заполнены. Фронтенд требовал выбора базы данных для КПВЭД, но бэкенд использует сервисную базу данных (serviceDB) и игнорирует параметр database.

## Решение
Сделали КПВЭД аналогично ОКПД2 - работает без выбора базы данных, так как данные хранятся в serviceDB и доступны сразу.

## Измененные файлы

### Frontend

#### 1. `frontend/app/classifiers/page.tsx`
- ✅ Убрана проверка `selectedDatabase` для КПВЭД в `useEffect`
- ✅ Обновлена функция `fetchStats` - убран параметр `database` из URL
- ✅ Обновлена функция `fetchHierarchy` - убрана проверка `selectedDatabase`
- ✅ Обновлена функция `searchKPVED` - убран параметр `database` из запроса
- ✅ Обновлена функция `loadKpved` - убрана проверка и передача `database`
- ✅ Обновлена функция `exportHierarchy` - убрана проверка `selectedDatabase`
- ✅ Скрыт селектор базы данных для КПВЭД
- ✅ Добавлено информационное сообщение, что КПВЭД использует сервисную БД

#### 2. `frontend/app/classifiers/[classifier]/page.tsx`
- ✅ Убрана проверка `selectedDatabase` для КПВЭД в `useEffect`
- ✅ Обновлена функция `fetchStats` - убран параметр `database` из URL
- ✅ Обновлена функция `fetchHierarchy` - убрана проверка и параметр `database`
- ✅ Обновлена функция `handleSearch` - убрана проверка и параметр `database`
- ✅ Скрыт селектор базы данных для КПВЭД
- ✅ Добавлено информационное сообщение для КПВЭД

#### 3. `frontend/app/api/kpved/hierarchy/route.ts`
- ✅ Убран параметр `database` из запроса к бэкенду

#### 4. `frontend/app/api/kpved/stats/route.ts`
- ✅ Убран параметр `database` из запроса к бэкенду

#### 5. `frontend/app/api/kpved/search/route.ts`
- ✅ Убран параметр `database` из запроса к бэкенду

#### 6. `frontend/app/classifiers/hooks/useKPVEDTree.ts`
- ✅ Убраны параметры `database` из всех функций:
  - `loadRootHierarchy()`
  - `loadChildNodes(parentCode)`
  - `toggleNode(code)`
  - `searchKPVED(query)`
  - `loadStats()`

#### 7. `frontend/lib/validation.ts`
- ✅ Удален неиспользуемый `database_id` из схемы `kpvedLoadSchema`

### Backend
Бэкенд уже был правильно настроен - использует `serviceDB` для всех операций с КПВЭД:
- `handleKpvedHierarchy` - использует `s.serviceDB.GetDB()`
- `handleKpvedSearch` - использует `s.serviceDB.GetDB()`
- `handleKpvedStats` - использует `s.serviceDB.GetDB()`
- `handleKpvedLoad` - использует `s.serviceDB` (игнорирует параметр `database` если передан)

## Результат

✅ КПВЭД теперь работает аналогично ОКПД2
- Данные автоматически загружаются при открытии страницы
- Не требуется выбор базы данных
- Работает с сервисной БД (serviceDB)
- Имеет понятные информационные сообщения

✅ Все упоминания `database` с КПВЭД в папке `frontend/app/classifiers` удалены

✅ Нет ошибок линтера

✅ Код стал более консистентным и понятным

## Тестирование

Рекомендуется проверить:
1. ✅ Открыть страницу `/classifiers/kpved` - классификатор должен загружаться автоматически
2. ✅ Проверить поиск по КПВЭД
3. ✅ Проверить загрузку дочерних узлов при раскрытии дерева
4. ✅ Проверить загрузку классификатора из файла
5. ✅ Проверить статистику - должна отображаться автоматически

## Примечания

- `frontend/app/results/page.tsx` - упоминания `selectedDatabase` оставлены, так как это другая функциональность (фильтрация результатов нормализации, а не просмотр классификатора)
- `frontend/app/api/kpved/reclassify-hierarchical/route.ts` - использует `database_id` для переклассификации нормализованных данных, это другой функционал

