package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

// TodoTask представляет задачу TODO
type TodoTask struct {
	ID            string    `json:"id"`
	File          string    `json:"file"`
	Line          int       `json:"line"`
	Type          string    `json:"type"`
	Priority      string    `json:"priority"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	AssignedTo    string    `json:"assignedTo,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	EstimatedHours int      `json:"estimatedHours"`
}

// TodoDB представляет базу данных задач
type TodoDB struct {
	Tasks    []TodoTask `json:"tasks"`
	LastScan *time.Time `json:"lastScan"`
	Version  string     `json:"version"`
}

func main() {
	dbPath := ".todos/tasks.json"
	
	// Загружаем БД
	data, err := os.ReadFile(dbPath)
	if err != nil {
		log.Fatalf("Ошибка чтения БД: %v", err)
	}

	var db TodoDB
	if err := json.Unmarshal(data, &db); err != nil {
		log.Fatalf("Ошибка парсинга БД: %v", err)
	}

	// Статистика
	total := len(db.Tasks)
	open := 0
	inProgress := 0
	resolved := 0
	critical := 0
	high := 0
	medium := 0
	low := 0

	// Группируем по приоритетам
	criticalTasks := []TodoTask{}
	highTasks := []TodoTask{}
	_ = []TodoTask{} // mediumTasks - зарезервировано для будущего использования
	_ = []TodoTask{} // lowTasks - зарезервировано для будущего использования

	for _, task := range db.Tasks {
		switch task.Status {
		case "OPEN":
			open++
		case "IN_PROGRESS":
			inProgress++
		case "RESOLVED", "TESTING":
			resolved++
		}

		switch task.Priority {
		case "CRITICAL":
			critical++
			if task.Status == "OPEN" {
				criticalTasks = append(criticalTasks, task)
			}
		case "HIGH":
			high++
			if task.Status == "OPEN" && len(highTasks) < 10 {
				highTasks = append(highTasks, task)
			}
		case "MEDIUM":
			medium++
		case "LOW":
			low++
		}
	}

	// Сортируем критические задачи
	sort.Slice(criticalTasks, func(i, j int) bool {
		return criticalTasks[i].CreatedAt.Before(criticalTasks[j].CreatedAt)
	})

	// Генерируем Markdown отчет
	reportMD := generateMarkdownReport(db, total, open, inProgress, resolved, critical, high, medium, low, criticalTasks, highTasks)
	
	// Сохраняем Markdown
	if err := os.WriteFile("TODO_REPORT.md", []byte(reportMD), 0644); err != nil {
		log.Fatalf("Ошибка записи Markdown: %v", err)
	}

	// Генерируем HTML отчет
	reportHTML := generateHTMLReport(db, total, open, inProgress, resolved, critical, high, medium, low, criticalTasks, highTasks)
	
	// Сохраняем HTML
	if err := os.WriteFile(".todos/dashboard.html", []byte(reportHTML), 0644); err != nil {
		log.Fatalf("Ошибка записи HTML: %v", err)
	}

	fmt.Println("✅ Отчеты сгенерированы:")
	fmt.Println("   - TODO_REPORT.md")
	fmt.Println("   - .todos/dashboard.html")
}

func generateMarkdownReport(db TodoDB, total, open, inProgress, resolved, critical, high, medium, low int, criticalTasks, highTasks []TodoTask) string {
	report := fmt.Sprintf(`# 🎯 Automated TODO Report

**Generated:** %s
**Last Scan:** %s

## 📈 Quick Stats

- **Total Tasks:** %d
- **Open Tasks:** %d
- **In Progress:** %d
- **Resolved:** %d
- **Completion Rate:** %.1f%%

## 🚨 Priority Distribution

- **CRITICAL:** %d
- **HIGH:** %d
- **MEDIUM:** %d
- **LOW:** %d

## 🚨 Critical Tasks (Need Immediate Attention)

`,
		time.Now().Format("2006-01-02 15:04:05"),
		formatTime(db.LastScan),
		total, open, inProgress, resolved,
		float64(resolved)*100/float64(total),
		critical, high, medium, low,
	)

	if len(criticalTasks) == 0 {
		report += "✅ No critical tasks!\n\n"
	} else {
		for _, task := range criticalTasks {
			report += fmt.Sprintf(`### %s:%d

- **Type:** %s
- **Description:** %s
- **Assigned to:** %s
- **Created:** %s
- **Estimated:** %d hours

`,
				task.File, task.Line,
				task.Type,
				task.Description,
				task.AssignedTo,
				task.CreatedAt.Format("2006-01-02"),
				task.EstimatedHours,
			)
		}
	}

	report += "## 🔴 High Priority Tasks\n\n"
	if len(highTasks) == 0 {
		report += "✅ No high priority tasks!\n\n"
	} else {
		for _, task := range highTasks {
			report += fmt.Sprintf("- **%s:%d** - %s (Assigned: %s)\n",
				task.File, task.Line, task.Description, task.AssignedTo)
		}
	}

	report += "\n## 🎯 Next Actions\n\n"
	report += "1. Review critical tasks\n"
	report += "2. Assign unassigned tasks\n"
	report += "3. Update status of in-progress tasks\n"
	report += "4. Plan sprint based on priorities\n\n"
	report += "[View Full Dashboard](./.todos/dashboard.html)\n"

	return report
}

func generateHTMLReport(db TodoDB, total, open, inProgress, resolved, critical, high, medium, low int, criticalTasks, highTasks []TodoTask) string {
	htmlTemplate := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TODO Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            padding: 20px;
        }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .stat-card h3 { color: #666; font-size: 14px; margin-bottom: 10px; }
        .stat-card .value { font-size: 32px; font-weight: bold; }
        .stat-card.critical .value { color: #dc3545; }
        .stat-card.high .value { color: #fd7e14; }
        .stat-card.medium .value { color: #ffc107; }
        .stat-card.low .value { color: #28a745; }
        .tasks-section {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        .tasks-section h2 { margin-bottom: 20px; color: #333; }
        .task-item {
            padding: 15px;
            border-left: 4px solid #ddd;
            margin-bottom: 10px;
            background: #f9f9f9;
        }
        .task-item.critical { border-left-color: #dc3545; }
        .task-item.high { border-left-color: #fd7e14; }
        .task-item.medium { border-left-color: #ffc107; }
        .task-item.low { border-left-color: #28a745; }
        .task-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 5px;
        }
        .task-file { font-weight: bold; color: #666; }
        .task-type { color: #999; font-size: 12px; }
        .task-desc { color: #333; margin: 5px 0; }
        .task-meta { font-size: 12px; color: #999; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎯 TODO Dashboard</h1>
        
        <div class="stats-grid">
            <div class="stat-card">
                <h3>Всего задач</h3>
                <div class="value">{{.Total}}</div>
            </div>
            <div class="stat-card">
                <h3>Открытых</h3>
                <div class="value">{{.Open}}</div>
            </div>
            <div class="stat-card">
                <h3>В работе</h3>
                <div class="value">{{.InProgress}}</div>
            </div>
            <div class="stat-card">
                <h3>Завершено</h3>
                <div class="value">{{.Resolved}}</div>
            </div>
            <div class="stat-card critical">
                <h3>Критических</h3>
                <div class="value">{{.Critical}}</div>
            </div>
            <div class="stat-card high">
                <h3>Высокий приоритет</h3>
                <div class="value">{{.High}}</div>
            </div>
            <div class="stat-card medium">
                <h3>Средний</h3>
                <div class="value">{{.Medium}}</div>
            </div>
            <div class="stat-card low">
                <h3>Низкий</h3>
                <div class="value">{{.Low}}</div>
            </div>
        </div>

        <div class="tasks-section">
            <h2>🚨 Critical Tasks</h2>
            {{range .CriticalTasks}}
            <div class="task-item critical">
                <div class="task-header">
                    <span class="task-file">{{.File}}:{{.Line}}</span>
                    <span class="task-type">{{.Type}}</span>
                </div>
                <div class="task-desc">{{.Description}}</div>
                <div class="task-meta">
                    Assigned: {{.AssignedTo}} | Created: {{.CreatedAt.Format "2006-01-02"}} | Est: {{.EstimatedHours}}h
                </div>
            </div>
            {{else}}
            <p>✅ No critical tasks!</p>
            {{end}}
        </div>

        <div class="tasks-section">
            <h2>🔴 High Priority Tasks</h2>
            {{range .HighTasks}}
            <div class="task-item high">
                <div class="task-header">
                    <span class="task-file">{{.File}}:{{.Line}}</span>
                    <span class="task-type">{{.Type}}</span>
                </div>
                <div class="task-desc">{{.Description}}</div>
                <div class="task-meta">
                    Assigned: {{.AssignedTo}} | Created: {{.CreatedAt.Format "2006-01-02"}}
                </div>
            </div>
            {{else}}
            <p>✅ No high priority tasks!</p>
            {{end}}
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("dashboard").Parse(htmlTemplate)
	if err != nil {
		log.Fatalf("Ошибка парсинга шаблона: %v", err)
	}

	type ReportData struct {
		Total, Open, InProgress, Resolved int
		Critical, High, Medium, Low        int
		CriticalTasks, HighTasks          []TodoTask
	}

	data := ReportData{
		Total:        total,
		Open:         open,
		InProgress:   inProgress,
		Resolved:     resolved,
		Critical:     critical,
		High:         high,
		Medium:       medium,
		Low:          low,
		CriticalTasks: criticalTasks,
		HighTasks:    highTasks,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatalf("Ошибка выполнения шаблона: %v", err)
	}

	return buf.String()
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

