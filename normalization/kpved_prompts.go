package normalization

import (
	"fmt"
	"strings"
)

// ClassificationPrompt промпт для классификации
type ClassificationPrompt struct {
	System string
	User   string
}

// PromptBuilder строитель промптов для классификации
type PromptBuilder struct {
	tree *KpvedTree
}

// NewPromptBuilder создает новый строитель промптов
func NewPromptBuilder(tree *KpvedTree) *PromptBuilder {
	return &PromptBuilder{tree: tree}
}

// BuildLevelPrompt строит промпт для указанного уровня
func (pb *PromptBuilder) BuildLevelPrompt(
	normalizedName string,
	category string,
	level KpvedLevel,
	candidates []*KpvedNode,
) *ClassificationPrompt {
	switch level {
	case LevelSection:
		return pb.buildSectionPrompt(normalizedName, category, candidates)
	case LevelClass:
		return pb.buildClassPrompt(normalizedName, category, candidates)
	case LevelSubclass:
		return pb.buildSubclassPrompt(normalizedName, category, candidates)
	case LevelGroup:
		return pb.buildGroupPrompt(normalizedName, category, candidates)
	default:
		return pb.buildSectionPrompt(normalizedName, category, candidates)
	}
}

// buildSectionPrompt строит промпт для уровня секций
func (pb *PromptBuilder) buildSectionPrompt(
	normalizedName string,
	category string,
	candidates []*KpvedNode,
) *ClassificationPrompt {
	// Формируем список секций
	var sectionsText strings.Builder
	for _, candidate := range candidates {
		sectionsText.WriteString(fmt.Sprintf("- %s: %s\n", candidate.Code, candidate.Name))
	}

	systemPrompt := fmt.Sprintf(`Ты - эксперт по классификации товаров по КПВЭД.

Выбери ОДИН наиболее подходящий раздел КПВЭД для товара.

Доступные разделы:
%s

Правила:
1. Выбери только ОДИН раздел
2. Учитывай тип товара и его назначение
3. Если товар может относиться к нескольким разделам, выбери наиболее специфичный

Формат ответа - ТОЛЬКО JSON (без markdown кода):
{
    "selected_code": "код раздела (одна буква)",
    "confidence": 0.95,
    "reasoning": "краткое (одно предложение) объяснение выбора"
}`, sectionsText.String())

	userPrompt := fmt.Sprintf("Товар: %s\nКатегория: %s", normalizedName, category)

	return &ClassificationPrompt{
		System: systemPrompt,
		User:   userPrompt,
	}
}

// buildClassPrompt строит промпт для уровня классов
func (pb *PromptBuilder) buildClassPrompt(
	normalizedName string,
	category string,
	candidates []*KpvedNode,
) *ClassificationPrompt {
	if len(candidates) == 0 {
		return &ClassificationPrompt{}
	}

	// Получаем название секции
	parentCode := candidates[0].ParentCode
	parentNode, _ := pb.tree.GetNode(parentCode)
	sectionName := "неизвестный раздел"
	if parentNode != nil {
		sectionName = parentNode.Name
	}

	// Формируем список классов (ограничиваем до 30 для краткости)
	var classesText strings.Builder
	maxClasses := 30
	for i, candidate := range candidates {
		if i >= maxClasses {
			classesText.WriteString(fmt.Sprintf("... и еще %d классов\n", len(candidates)-maxClasses))
			break
		}
		classesText.WriteString(fmt.Sprintf("- %s: %s\n", candidate.Code, candidate.Name))
	}

	systemPrompt := fmt.Sprintf(`Ты - эксперт по классификации товаров по КПВЭД.

Выбери ОДИН наиболее подходящий класс в разделе "%s".

Доступные классы:
%s

Правила:
1. Выбери только ОДИН класс
2. Класс должен точно соответствовать типу товара
3. Учитывай специфику и назначение товара

Формат ответа - ТОЛЬКО JSON (без markdown кода):
{
    "selected_code": "код класса (две цифры)",
    "confidence": 0.90,
    "reasoning": "почему именно этот класс"
}`, sectionName, classesText.String())

	userPrompt := fmt.Sprintf("Товар: %s\nКатегория: %s", normalizedName, category)

	return &ClassificationPrompt{
		System: systemPrompt,
		User:   userPrompt,
	}
}

// buildSubclassPrompt строит промпт для уровня подклассов
func (pb *PromptBuilder) buildSubclassPrompt(
	normalizedName string,
	category string,
	candidates []*KpvedNode,
) *ClassificationPrompt {
	if len(candidates) == 0 {
		return &ClassificationPrompt{}
	}

	// Получаем название класса
	parentCode := candidates[0].ParentCode
	parentNode, _ := pb.tree.GetNode(parentCode)
	className := "неизвестный класс"
	if parentNode != nil {
		className = parentNode.Name
	}

	// Формируем список подклассов (ограничиваем до 25)
	var subclassesText strings.Builder
	maxSubclasses := 25
	for i, candidate := range candidates {
		if i >= maxSubclasses {
			subclassesText.WriteString(fmt.Sprintf("... и еще %d подклассов\n", len(candidates)-maxSubclasses))
			break
		}
		subclassesText.WriteString(fmt.Sprintf("- %s: %s\n", candidate.Code, candidate.Name))
	}

	systemPrompt := fmt.Sprintf(`Ты - эксперт по классификации товаров по КПВЭД.

Выбери ОДИН наиболее подходящий подкласс в классе "%s".

Доступные подклассы:
%s

Правила:
1. Выбери только ОДИН подкласс
2. Подкласс должен максимально точно описывать товар
3. Учитывай материал, назначение и особенности товара

Формат ответа - ТОЛЬКО JSON (без markdown кода):
{
    "selected_code": "код подкласса (формат XX.Y)",
    "confidence": 0.85,
    "reasoning": "почему именно этот подкласс"
}`, className, subclassesText.String())

	userPrompt := fmt.Sprintf("Товар: %s\nКатегория: %s", normalizedName, category)

	return &ClassificationPrompt{
		System: systemPrompt,
		User:   userPrompt,
	}
}

// buildGroupPrompt строит промпт для уровня групп
func (pb *PromptBuilder) buildGroupPrompt(
	normalizedName string,
	category string,
	candidates []*KpvedNode,
) *ClassificationPrompt {
	if len(candidates) == 0 {
		return &ClassificationPrompt{}
	}

	// Получаем название подкласса
	parentCode := candidates[0].ParentCode
	parentNode, _ := pb.tree.GetNode(parentCode)
	subclassName := "неизвестный подкласс"
	if parentNode != nil {
		subclassName = parentNode.Name
	}

	// Формируем список групп (ограничиваем до 20)
	var groupsText strings.Builder
	maxGroups := 20
	for i, candidate := range candidates {
		if i >= maxGroups {
			groupsText.WriteString(fmt.Sprintf("... и еще %d групп\n", len(candidates)-maxGroups))
			break
		}
		groupsText.WriteString(fmt.Sprintf("- %s: %s\n", candidate.Code, candidate.Name))
	}

	systemPrompt := fmt.Sprintf(`Ты - эксперт по классификации товаров по КПВЭД.

Выбери ОДИН наиболее подходящий код группы в подклассе "%s".

Доступные группы:
%s

Правила:
1. Выбери только ОДИН код
2. Код должен максимально точно соответствовать товару
3. Это финальный уровень классификации - будь максимально точен

Формат ответа - ТОЛЬКО JSON (без markdown кода):
{
    "selected_code": "код группы (формат XX.YY или XX.YY.Z)",
    "confidence": 0.80,
    "reasoning": "финальное обоснование выбора"
}`, subclassName, groupsText.String())

	userPrompt := fmt.Sprintf("Товар: %s\nКатегория: %s", normalizedName, category)

	return &ClassificationPrompt{
		System: systemPrompt,
		User:   userPrompt,
	}
}

// GetPromptSize возвращает примерный размер промпта в байтах
func (p *ClassificationPrompt) GetPromptSize() int {
	return len(p.System) + len(p.User)
}

// FormatForAPI форматирует промпт для отправки в API
func (p *ClassificationPrompt) FormatForAPI() map[string]string {
	return map[string]string{
		"system": p.System,
		"user":   p.User,
	}
}
