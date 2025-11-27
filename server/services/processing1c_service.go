package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	apperrors "httpserver/server/errors"
)

// Processing1CService сервис для работы с обработками 1С
type Processing1CService struct {
	workDir string
}

// NewProcessing1CService создает новый сервис для работы с обработками 1С
func NewProcessing1CService() *Processing1CService {
	workDir, _ := os.Getwd()
	return &Processing1CService{
		workDir: workDir,
	}
}

// GenerateProcessingXML генерирует XML файл обработки 1С
func (s *Processing1CService) GenerateProcessingXML() (string, error) {
	// Читаем файлы модулей
	modulePath := filepath.Join(s.workDir, "1c_processing", "Module", "Module.bsl")
	moduleCode, err := os.ReadFile(modulePath)
	if err != nil {
		return "", apperrors.NewInternalError("не удалось прочитать Module.bsl", err)
	}

	extensionsPath := filepath.Join(s.workDir, "1c_module_extensions.bsl")
	extensionsCode, err := os.ReadFile(extensionsPath)
	if err != nil {
		extensionsCode = []byte("")
	}

	exportFunctionsPath := filepath.Join(s.workDir, "1c_export_functions.txt")
	exportFunctionsCode, err := os.ReadFile(exportFunctionsPath)
	if err != nil {
		exportFunctionsCode = []byte("")
	}

	// Объединяем код модуля
	fullModuleCode := string(moduleCode)

	// Добавляем код из export_functions
	if len(exportFunctionsCode) > 0 {
		exportCodeStr := string(exportFunctionsCode)
		startMarker := "#Область ПрограммныйИнтерфейс"
		endMarker := "#КонецОбласти"

		startPos := strings.Index(exportCodeStr, startMarker)
		if startPos >= 0 {
			endPos := strings.Index(exportCodeStr[startPos+len(startMarker):], endMarker)
			if endPos >= 0 {
				endPos += startPos + len(startMarker)
				programInterfaceCode := exportCodeStr[startPos : endPos+len(endMarker)]
				fullModuleCode += "\n\n" + programInterfaceCode
			}
		}
	}

	// Добавляем расширения
	if len(extensionsCode) > 0 {
		fullModuleCode += "\n\n" + string(extensionsCode)
	}

	// Генерируем UUID для обработки
	processingUUID := strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", ""))

	// TODO: Генерация полного XML файла обработки
	// Это требует большого объема кода из handle1CProcessingXML
	xmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:app="http://v8.1c.ru/8.1/data/enterprise/current-config" xmlns:cfg="http://v8.1c.ru/8.1/data/ui" xmlns:cmi="http://v8.1c.ru/8.1/data/core" xmlns:ent="http://v8.1c.ru/8.1/data/enterprise" xmlns:lf="http://v8.1c.ru/8.2/managers/logform/field" xmlns:style="http://v8.1c.ru/8.1/data/ui/style" xmlns:sys="http://v8.1c.ru/8.1/data/ui/fonts/system" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:v8ui="http://v8.1c.ru/8.1/data/ui" xmlns:web="http://v8.1c.ru/8.1/data/ui/colors/web" xmlns:win="http://v8.1c.ru/8.1/data/ui/colors/windows" xmlns:xpr="http://v8.1c.ru/8.3/xcf/prepare" xmlns:xr="http://v8.1c.ru/8.3/xcf" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="2.14">
	<DataProcessor uuid="%s">
		<Properties>
			<Name>ОбработкаЗагрузкиДанных</Name>
			<Synonym>
				<Key>ru</Key>
				<Value>Обработка загрузки данных</Value>
			</Synonym>
		</Properties>
		<ChildObjects>
			<ObjectModule>
				<Content>%s</Content>
			</ObjectModule>
		</ChildObjects>
	</DataProcessor>
</MetaDataObject>`, processingUUID, fullModuleCode)

	return xmlContent, nil
}

