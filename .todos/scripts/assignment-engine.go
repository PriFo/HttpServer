package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// TodoTask представляет задачу TODO
type TodoTask struct {
	ID           string   `json:"id"`
	File         string   `json:"file"`
	Line         int      `json:"line"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Priority     string   `json:"priority"`
	Status       string   `json:"status"`
	AssignedTo   string   `json:"assignedTo"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
	FileType     string   `json:"fileType"`
	EstimatedHours int    `json:"estimatedHours"`
	Dependencies []string `json:"dependencies"`
	RelatedFiles []string `json:"relatedFiles"`
}

// Developer представляет разработчика
type Developer struct {
	Name      string
	Team      string
	Skills    []string
	Workload  int
	Specialty string
}

// TeamConfig конфигурация команды
type TeamConfig struct {
	Team       map[string][]string `json:"team"`
	Specialties map[string][]string `json:"specialties"`
	Workload   map[string]int      `json:"workload"`
}

// AssignmentEngine движок для автоназначения задач
type AssignmentEngine struct {
	TeamConfig TeamConfig
	Developers []Developer
}

// NewAssignmentEngine создает новый движок назначения
func NewAssignmentEngine(configPath string) (*AssignmentEngine, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read team config: %w", err)
	}

	var config TeamConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse team config: %w", err)
	}

	// Создаем список разработчиков из конфига
	developers := make([]Developer, 0)
	for teamName, members := range config.Team {
		for _, member := range members {
			// Определяем специализацию
			specialty := determineSpecialty(member, teamName, config.Specialties)
			
			developers = append(developers, Developer{
				Name:      member,
				Team:      teamName,
				Skills:    getSkillsForSpecialty(specialty),
				Workload:  config.Workload[member],
				Specialty: specialty,
			})
		}
	}

	return &AssignmentEngine{
		TeamConfig: config,
		Developers: developers,
	}, nil
}

// AssignTask назначает задачу разработчику
func (e *AssignmentEngine) AssignTask(task TodoTask) string {
	// Определяем требуемые навыки на основе типа файла и задачи
	requiredSkills := e.getRequiredSkills(task)
	
	// Фильтруем разработчиков по навыкам
	candidates := e.filterBySkills(requiredSkills)
	
	if len(candidates) == 0 {
		// Если нет подходящих по навыкам, используем тип файла
		return e.assignByFileType(task)
	}
	
	// Сортируем по загруженности (меньше загруженные - выше)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Workload < candidates[j].Workload
	})
	
	// Выбираем наименее загруженного
	if len(candidates) > 0 {
		assigned := candidates[0].Name
		// Обновляем загруженность
		e.TeamConfig.Workload[assigned]++
		return assigned
	}
	
	return "unassigned"
}

// getRequiredSkills определяет требуемые навыки для задачи
func (e *AssignmentEngine) getRequiredSkills(task TodoTask) []string {
	skills := make([]string, 0)
	
	// Навыки на основе типа файла
	switch task.FileType {
	case "backend":
		skills = append(skills, "go", "python", "backend")
	case "frontend":
		skills = append(skills, "typescript", "react", "frontend")
	case "script", "devops":
		skills = append(skills, "bash", "docker", "devops")
	}
	
	// Навыки на основе расширения файла
	ext := getFileExtension(task.File)
	switch ext {
	case ".go":
		skills = append(skills, "go")
	case ".ts", ".tsx":
		skills = append(skills, "typescript")
	case ".js", ".jsx":
		skills = append(skills, "javascript")
	case ".py":
		skills = append(skills, "python")
	case ".sh":
		skills = append(skills, "bash")
	}
	
	return skills
}

// filterBySkills фильтрует разработчиков по навыкам
func (e *AssignmentEngine) filterBySkills(requiredSkills []string) []Developer {
	candidates := make([]Developer, 0)
	
	for _, dev := range e.Developers {
		hasSkills := false
		for _, skill := range requiredSkills {
			for _, devSkill := range dev.Skills {
				if strings.EqualFold(devSkill, skill) {
					hasSkills = true
					break
				}
			}
			if hasSkills {
				break
			}
		}
		
		if hasSkills {
			candidates = append(candidates, dev)
		}
	}
	
	return candidates
}

// assignByFileType назначает по типу файла
func (e *AssignmentEngine) assignByFileType(task TodoTask) string {
	switch task.FileType {
	case "backend":
		return "backend-team"
	case "frontend":
		return "frontend-team"
	case "script", "devops":
		return "devops"
	default:
		return "unassigned"
	}
}

// CalculateWorkload вычисляет текущую загруженность разработчика
func (e *AssignmentEngine) CalculateWorkload(developer string) int {
	return e.TeamConfig.Workload[developer]
}

// FindBestAssignee находит лучшего исполнителя для задачи
func (e *AssignmentEngine) FindBestAssignee(task TodoTask) string {
	return e.AssignTask(task)
}

// Вспомогательные функции

func determineSpecialty(member, team string, specialties map[string][]string) string {
	for tech, teams := range specialties {
		for _, t := range teams {
			if t == team {
				return tech
			}
		}
	}
	return team
}

func getSkillsForSpecialty(specialty string) []string {
	skillMap := map[string][]string{
		"go":         {"go", "backend", "golang"},
		"typescript": {"typescript", "javascript", "frontend", "react"},
		"react":      {"react", "typescript", "javascript", "frontend"},
		"python":     {"python", "backend"},
		"bash":       {"bash", "shell", "devops"},
		"docker":     {"docker", "devops", "containers"},
	}
	
	if skills, ok := skillMap[specialty]; ok {
		return skills
	}
	return []string{specialty}
}

func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return "." + parts[len(parts)-1]
	}
	return ""
}

// Main функция для тестирования
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: assignment-engine <team-config.json>")
		os.Exit(1)
	}
	
	engine, err := NewAssignmentEngine(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	
	// Пример использования
	task := TodoTask{
		File:     "server.go",
		Line:     42,
		FileType: "backend",
		Priority: "HIGH",
		Type:     "TODO",
	}
	
	assigned := engine.AssignTask(task)
	fmt.Printf("Task assigned to: %s\n", assigned)
}


