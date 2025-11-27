# 🔗 Интеграция системы TODO

## Git Hooks

### Pre-commit Hook

Автоматически проверяет наличие критических TODO перед коммитом.

**Локация:** `.git/hooks/pre-commit`

**Поведение:**
- Блокирует коммит при наличии критических задач
- Показывает список критических задач
- Можно обойти с `git commit --no-verify`

**Настройка:**
```bash
chmod +x .git/hooks/pre-commit
```

## CI/CD Integration

### GitHub Actions

```yaml
name: TODO Check

on:
  pull_request:
    branches: [ main ]

jobs:
  check-todos:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y jq
      - name: Scan TODOs
        run: |
          bash .todos/scripts/scan-todos.sh .
      - name: Check critical TODOs
        run: |
          CRITICAL=$(jq '[.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN")] | length' .todos/tasks.json)
          if [ "$CRITICAL" -gt 0 ]; then
            echo "Found $CRITICAL critical TODOs"
            exit 1
          fi
```

### GitLab CI

```yaml
check-todos:
  stage: test
  script:
    - apt-get update && apt-get install -y jq
    - bash .todos/scripts/scan-todos.sh .
    - |
      CRITICAL=$(jq '[.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN")] | length' .todos/tasks.json)
      if [ "$CRITICAL" -gt 0 ]; then
        echo "Found $CRITICAL critical TODOs"
        exit 1
      fi
```

## IDE Integration

### VS Code

Добавьте в `.vscode/tasks.json`:
```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Scan TODOs",
      "type": "shell",
      "command": "npm run todos:scan",
      "group": "build",
      "presentation": {
        "reveal": "always"
      }
    }
  ]
}
```

### Cursor

Система уже интегрирована через `.cursorrules`.

Используйте команды:
- `npm run todos:scan`
- `npm run todos:report`
- `npm run todos:stats`

## Slack Integration

Создайте скрипт для отправки уведомлений:

```bash
#!/bin/bash
# .todos/scripts/slack-notify.sh

WEBHOOK_URL="${SLACK_WEBHOOK_URL}"
CRITICAL=$(jq '[.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN")] | length' .todos/tasks.json)

if [ "$CRITICAL" -gt 0 ]; then
  curl -X POST -H 'Content-type: application/json' \
    --data "{\"text\":\"🚨 Found $CRITICAL critical TODOs!\"}" \
    "$WEBHOOK_URL"
fi
```

## Email Integration

```bash
#!/bin/bash
# .todos/scripts/email-report.sh

CRITICAL=$(jq '[.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN")] | length' .todos/tasks.json)

if [ "$CRITICAL" -gt 0 ]; then
  echo "Found $CRITICAL critical TODOs" | mail -s "Critical TODOs Alert" team@example.com
fi
```

## Scheduled Tasks

### Linux/Mac (Cron)

```bash
# Каждый час
0 * * * * cd /path/to/project && npm run todos:scan

# Ежедневный отчет в 9:00
0 9 * * * cd /path/to/project && npm run todos:report

# Еженедельная очистка
0 0 * * 0 cd /path/to/project && npm run todos:cleanup 30
```

### Windows (Task Scheduler)

```powershell
# Создать задачу
$action = New-ScheduledTaskAction -Execute "npm" -Argument "run todos:scan" -WorkingDirectory "E:\HttpServer"
$trigger = New-ScheduledTaskTrigger -Daily -At "09:00"
Register-ScheduledTask -TaskName "TODO Scanner" -Action $action -Trigger $trigger
```

## API Integration

Система может быть интегрирована через JSON API:

```javascript
// Чтение задач
const tasks = require('.todos/tasks.json');

// Фильтрация
const critical = tasks.tasks.filter(t => 
  t.priority === 'CRITICAL' && t.status === 'OPEN'
);

// Обновление статуса
tasks.tasks = tasks.tasks.map(t => 
  t.id === taskId ? { ...t, status: 'RESOLVED' } : t
);
```

## Webhook Integration

Создайте endpoint для получения уведомлений:

```javascript
// webhook.js
const fs = require('fs');
const tasks = JSON.parse(fs.readFileSync('.todos/tasks.json'));

app.post('/webhook/todos', (req, res) => {
  const critical = tasks.tasks.filter(t => 
    t.priority === 'CRITICAL' && t.status === 'OPEN'
  );
  
  // Отправить уведомление
  sendNotification(critical);
  
  res.json({ critical: critical.length });
});
```

## Database Integration

Экспортируйте задачи в базу данных:

```bash
# PostgreSQL
node -e "
  const tasks = require('.todos/tasks.json');
  // SQL insert statements
  tasks.tasks.forEach(task => {
    console.log(\`INSERT INTO todos VALUES (...);\`);
  });
" | psql database
```

## Monitoring Integration

Интеграция с системами мониторинга:

```bash
# Prometheus metrics
echo "todo_critical_count $(jq '[.tasks[] | select(.priority == "CRITICAL")] | length' .todos/tasks.json)"
echo "todo_total_count $(jq '.tasks | length' .todos/tasks.json)"
```

