package database

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DatabaseFilenameInfo содержит информацию, извлеченную из имени файла базы данных
type DatabaseFilenameInfo struct {
	DisplayName    string // Читаемое название (например, "ERP WE Номенклатура")
	ConfigName     string // Название конфигурации 1С (например, "ERPWE")
	DatabaseType   string // Тип базы данных (Номенклатура или Контрагенты)
	DataType       string // Тип данных для project_type (nomenclature, counterparties)
}

// ParseDatabaseNameFromFilename извлекает читаемое название из имени файла базы данных
// Примеры:
// "Выгрузка_Номенклатура_ERPWE_Unknown_Unknown_2025_11_20_10_18_55.db" -> "ERP WE Номенклатура"
// "Выгрузка_Контрагенты_БухгалтерияДляКазахстана_Unknown_Unknown_2025.db" -> "БухгалтерияДляКазахстана Контрагенты"
func ParseDatabaseNameFromFilename(fileName string) string {
	info := ParseDatabaseFileInfo(fileName)
	return info.DisplayName
}

// ParseDatabaseFileInfo извлекает полную информацию из имени файла базы данных
func ParseDatabaseFileInfo(fileName string) DatabaseFilenameInfo {
	info := DatabaseFilenameInfo{
		DisplayName:  fileName,
		ConfigName:   "",
		DatabaseType: "",
		DataType:     "",
	}

	// Убираем расширение
	nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	
	// Разбиваем по подчеркиваниям
	parts := strings.Split(nameWithoutExt, "_")
	
	if len(parts) < 3 {
		// Если формат не соответствует ожидаемому, возвращаем имя файла без расширения
		info.DisplayName = nameWithoutExt
		return info
	}
	
	// Формат: Выгрузка_<Тип>_<Конфигурация>_...
	// Тип: Номенклатура или Контрагенты
	// Конфигурация: например, ERPWE, БухгалтерияДляКазахстана
	
	dbType := parts[1] // Номенклатура или Контрагенты
	configName := parts[2] // Название конфигурации
	
	// Если конфигурация "Unknown", пробуем взять следующую часть
	if configName == "Unknown" && len(parts) > 3 {
		configName = parts[3]
	}
	
	// Сохраняем исходное название конфигурации
	info.ConfigName = configName
	info.DatabaseType = dbType
	
	// Определяем тип данных для project_type
	if dbType == "Номенклатура" {
		info.DataType = "nomenclature"
	} else if dbType == "Контрагенты" {
		info.DataType = "counterparties"
	}
	
	// Формируем читаемое название
	var result strings.Builder
	
	// Добавляем название конфигурации, разделяя заглавные буквы пробелами
	// Например, "ERPWE" -> "ERP WE", "БухгалтерияДляКазахстана" -> "БухгалтерияДляКазахстана"
	if configName != "Unknown" && configName != "" {
		// Для латинских букв: разделяем по заглавным
		if strings.ContainsAny(configName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			var formattedConfig strings.Builder
			for i, r := range configName {
				if i > 0 && r >= 'A' && r <= 'Z' {
					formattedConfig.WriteRune(' ')
				}
				formattedConfig.WriteRune(r)
			}
			result.WriteString(formattedConfig.String())
		} else {
			result.WriteString(configName)
		}
		result.WriteString(" ")
	}
	
	// Добавляем тип
	result.WriteString(dbType)
	
	info.DisplayName = strings.TrimSpace(result.String())
	
	return info
}

// FindMatchingProjectForDatabase находит подходящий проект для базы данных на основе имени файла
// Возвращает проект, если найден подходящий, или nil если не найден
func FindMatchingProjectForDatabase(serviceDB *ServiceDB, clientID int, filePath string) (*ClientProject, error) {
	if serviceDB == nil {
		return nil, fmt.Errorf("serviceDB is nil")
	}

	fileName := filepath.Base(filePath)
	fileInfo := ParseDatabaseFileInfo(fileName)

	// Если не удалось определить тип данных, возвращаем nil
	if fileInfo.DataType == "" {
		return nil, nil
	}

	// Получаем все проекты клиента
	projects, err := serviceDB.GetClientProjects(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client projects: %w", err)
	}

	// Ищем проект с подходящим типом
	for _, project := range projects {
		// Точное совпадение типа
		if project.ProjectType == fileInfo.DataType {
			return project, nil
		}

		// Для nomenclature_counterparties принимаем и nomenclature, и counterparties
		if project.ProjectType == "nomenclature_counterparties" {
			if fileInfo.DataType == "nomenclature" || fileInfo.DataType == "counterparties" {
				return project, nil
			}
		}
	}

	return nil, nil
}

// IsCounterpartyProjectType проверяет, является ли тип проекта проектом контрагентов
// Возвращает true для типов "counterparty" и "nomenclature_counterparties"
func IsCounterpartyProjectType(projectType string) bool {
	return projectType == "counterparty" || projectType == "nomenclature_counterparties"
}

