package normalization

import (
	"fmt"
	"strings"

	"httpserver/database"
	"httpserver/extractors"
)

// CounterpartyDuplicateGroup группа дубликатов контрагентов
type CounterpartyDuplicateGroup struct {
	Key        string // Ключ группы (ИНН/КПП или БИН)
	KeyType    string // Тип ключа: "inn_kpp", "bin"
	Items      []*CounterpartyDuplicateItem
	MasterItem *CounterpartyDuplicateItem // Рекомендуемая основная запись
	Confidence float64                    // Уверенность в том, что это дубликаты (1.0 для ИНН/КПП и БИН)
}

// CounterpartyDuplicateItem элемент контрагента для анализа дублей
type CounterpartyDuplicateItem struct {
	ID                   int
	Reference            string
	Code                 string
	Name                 string
	Attributes           string // Сохраняем оригинальные атрибуты для извлечения данных
	INN                  string
	KPP                  string
	BIN                  string
	LegalAddress         string
	PostalAddress        string
	ContactPhone         string
	ContactEmail         string
	ContactPerson        string
	BankName             string
	BankAccount          string
	CorrespondentAccount string
	BIK                  string
	QualityScore         float64
	SourceDatabase       string
	EnrichmentApplied    bool
	SourceEnrichment     string
	DatabaseCount        int // Количество связанных баз данных
}

// CounterpartyDuplicateAnalyzer анализатор дублей контрагентов
type CounterpartyDuplicateAnalyzer struct {
}

// NewCounterpartyDuplicateAnalyzer создает новый анализатор дублей контрагентов
func NewCounterpartyDuplicateAnalyzer() *CounterpartyDuplicateAnalyzer {
	return &CounterpartyDuplicateAnalyzer{}
}

// AnalyzeDuplicates анализирует контрагентов на наличие дублей по ИНН/КПП и БИН
func (cda *CounterpartyDuplicateAnalyzer) AnalyzeDuplicates(counterparties []*database.CatalogItem) []CounterpartyDuplicateGroup {
	groups := []CounterpartyDuplicateGroup{}

	// 1. Группируем по связке ИНН/КПП
	innKppGroups := cda.groupByINNKPP(counterparties)
	groups = append(groups, innKppGroups...)

	// 2. Группируем по БИН
	binGroups := cda.groupByBIN(counterparties)
	groups = append(groups, binGroups...)

	// 3. Объединяем пересекающиеся группы (если у контрагента есть и ИНН/КПП, и БИН)
	mergedGroups := cda.mergeOverlappingGroups(groups)

	// 4. Выбираем master record для каждой группы
	for i := range mergedGroups {
		mergedGroups[i].MasterItem = cda.selectMasterRecord(mergedGroups[i].Items)
	}

	return mergedGroups
}

// groupByINNKPP группирует контрагентов по связке ИНН/КПП
func (cda *CounterpartyDuplicateAnalyzer) groupByINNKPP(counterparties []*database.CatalogItem) []CounterpartyDuplicateGroup {
	groups := []CounterpartyDuplicateGroup{}
	innKppMap := make(map[string][]*CounterpartyDuplicateItem)

	// Извлекаем данные и группируем
	for _, item := range counterparties {
		inn, _ := extractors.ExtractINNFromAttributes(item.Attributes)
		kpp, _ := extractors.ExtractKPPFromAttributes(item.Attributes)

		// Создаем ключ: ИНН/КПП или только ИНН
		var key string
		if inn != "" && kpp != "" {
			key = fmt.Sprintf("%s/%s", inn, kpp)
		} else if inn != "" {
			key = inn
		} else {
			continue // Пропускаем, если нет ИНН
		}

		// Извлекаем все данные из атрибутов
		duplicateItem := cda.catalogItemToDuplicateItem(item, inn, kpp, "")

		innKppMap[key] = append(innKppMap[key], duplicateItem)
	}

	// Создаем группы для дубликатов (только если больше 1 элемента)
	for key, items := range innKppMap {
		if len(items) > 1 {
			group := CounterpartyDuplicateGroup{
				Key:        key,
				KeyType:    "inn_kpp",
				Items:      items,
				Confidence: 1.0, // 100% уверенность для ИНН/КПП
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// groupByBIN группирует контрагентов по БИН
func (cda *CounterpartyDuplicateAnalyzer) groupByBIN(counterparties []*database.CatalogItem) []CounterpartyDuplicateGroup {
	groups := []CounterpartyDuplicateGroup{}
	binMap := make(map[string][]*CounterpartyDuplicateItem)

	// Извлекаем данные и группируем
	for _, item := range counterparties {
		bin, err := extractors.ExtractBINFromAttributes(item.Attributes)
		if err != nil || bin == "" {
			continue // Пропускаем, если нет БИН
		}

		// Извлекаем все данные из атрибутов
		duplicateItem := cda.catalogItemToDuplicateItem(item, "", "", bin)

		binMap[bin] = append(binMap[bin], duplicateItem)
	}

	// Создаем группы для дубликатов (только если больше 1 элемента)
	for bin, items := range binMap {
		if len(items) > 1 {
			group := CounterpartyDuplicateGroup{
				Key:        bin,
				KeyType:    "bin",
				Items:      items,
				Confidence: 1.0, // 100% уверенность для БИН
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// mergeOverlappingGroups объединяет пересекающиеся группы
// Например, если контрагент имеет и ИНН/КПП, и БИН, и они попадают в разные группы
func (cda *CounterpartyDuplicateAnalyzer) mergeOverlappingGroups(groups []CounterpartyDuplicateGroup) []CounterpartyDuplicateGroup {
	if len(groups) == 0 {
		return groups
	}

	merged := []CounterpartyDuplicateGroup{}
	processed := make(map[int]bool) // Отслеживаем обработанные группы

	for i, group1 := range groups {
		if processed[i] {
			continue
		}

		mergedGroup := group1
		processed[i] = true

		// Ищем пересекающиеся группы
		for j, group2 := range groups {
			if i == j || processed[j] {
				continue
			}

			// Проверяем, есть ли общие элементы
			if cda.hasCommonItems(group1.Items, group2.Items) {
				// Объединяем группы
				mergedGroup.Items = cda.mergeItems(mergedGroup.Items, group2.Items)
				// Объединяем ключи
				if mergedGroup.Key != group2.Key {
					mergedGroup.Key = fmt.Sprintf("%s|%s", mergedGroup.Key, group2.Key)
				}
				// Объединяем типы ключей
				if mergedGroup.KeyType != group2.KeyType {
					mergedGroup.KeyType = fmt.Sprintf("%s+%s", mergedGroup.KeyType, group2.KeyType)
				}
				processed[j] = true
			}
		}

		merged = append(merged, mergedGroup)
	}

	return merged
}

// hasCommonItems проверяет, есть ли общие элементы в двух группах
func (cda *CounterpartyDuplicateAnalyzer) hasCommonItems(items1, items2 []*CounterpartyDuplicateItem) bool {
	ids1 := make(map[int]bool)
	for _, item := range items1 {
		ids1[item.ID] = true
	}

	for _, item := range items2 {
		if ids1[item.ID] {
			return true
		}
	}

	return false
}

// mergeItems объединяет два списка элементов, убирая дубликаты
func (cda *CounterpartyDuplicateAnalyzer) mergeItems(items1, items2 []*CounterpartyDuplicateItem) []*CounterpartyDuplicateItem {
	merged := make(map[int]*CounterpartyDuplicateItem)

	for _, item := range items1 {
		merged[item.ID] = item
	}

	for _, item := range items2 {
		if _, exists := merged[item.ID]; !exists {
			merged[item.ID] = item
		}
	}

	result := make([]*CounterpartyDuplicateItem, 0, len(merged))
	for _, item := range merged {
		result = append(result, item)
	}

	return result
}

// selectMasterRecord выбирает основную запись из группы дубликатов
// Критерии выбора:
// 1. Наибольшая полнота данных (наличие адреса, контактов)
// 2. Наибольший quality_score
// 3. Наиболее полное название
func (cda *CounterpartyDuplicateAnalyzer) selectMasterRecord(items []*CounterpartyDuplicateItem) *CounterpartyDuplicateItem {
	if len(items) == 0 {
		return nil
	}

	if len(items) == 1 {
		return items[0]
	}

	var bestItem *CounterpartyDuplicateItem
	bestScore := -1.0

	for _, item := range items {
		score := cda.calculateMasterScore(item)
		if score > bestScore {
			bestScore = score
			bestItem = item
		}
	}

	return bestItem
}

// SelectMasterRecordWithStrategy выбирает master record с учетом стратегии
func (cda *CounterpartyDuplicateAnalyzer) SelectMasterRecordWithStrategy(items []*CounterpartyDuplicateItem, strategy string) *CounterpartyDuplicateItem {
	if len(items) == 0 {
		return nil
	}

	if len(items) == 1 {
		return items[0]
	}

	switch strategy {
	case "max_quality":
		// Выбираем запись с максимальным качеством
		var bestItem *CounterpartyDuplicateItem
		bestQuality := -1.0
		for _, item := range items {
			if item.QualityScore > bestQuality {
				bestQuality = item.QualityScore
				bestItem = item
			}
		}
		return bestItem
	case "max_databases":
		// Выбираем запись из базы с максимальным количеством связанных баз
		var bestItem *CounterpartyDuplicateItem
		maxDatabases := -1
		for _, item := range items {
			if item.DatabaseCount > maxDatabases {
				maxDatabases = item.DatabaseCount
				bestItem = item
			}
		}
		return bestItem
	case "max_data":
		fallthrough
	default:
		// Используем стандартную логику (max_data)
		return cda.selectMasterRecord(items)
	}
}

// calculateMasterScore вычисляет оценку пригодности записи как master record
func (cda *CounterpartyDuplicateAnalyzer) calculateMasterScore(item *CounterpartyDuplicateItem) float64 {
	score := 0.0

	// Наличие ИНН/КПП/БИН (обязательно)
	if item.INN != "" || item.BIN != "" {
		score += 30.0
	}

	// Наличие КПП (дополнительно)
	if item.KPP != "" {
		score += 10.0
	}

	// Количество заполненных полей (чем больше, тем лучше)
	filledFieldsCount := cda.countFilledFields(item)
	score += float64(filledFieldsCount) * 3.0 // 3 балла за каждое заполненное поле

	// Наличие адреса
	if item.LegalAddress != "" {
		score += 15.0
	}
	if item.PostalAddress != "" {
		score += 10.0
	}

	// Контактная информация
	if item.ContactPhone != "" {
		score += 5.0
	}
	if item.ContactEmail != "" {
		score += 5.0
	}
	if item.ContactPerson != "" {
		score += 5.0
	}

	// Банковские реквизиты
	if item.BankName != "" {
		score += 5.0
	}
	if item.BankAccount != "" {
		score += 5.0
	}
	if item.CorrespondentAccount != "" {
		score += 3.0
	}
	if item.BIK != "" {
		score += 3.0
	}

	// Количество связанных баз данных (больше баз = больше данных)
	score += float64(item.DatabaseCount) * 5.0

	// Наличие обогащенных данных
	if item.EnrichmentApplied {
		score += 20.0
		// Приоритет источника обогащения
		enrichmentPriority := cda.getEnrichmentPriority(item.SourceEnrichment)
		score += enrichmentPriority * 10.0
	}

	// Полнота названия (длина и наличие организационно-правовой формы)
	name := strings.TrimSpace(item.Name)
	if len(name) > 10 {
		score += 10.0
	}
	// Проверка на наличие ОПФ
	opfKeywords := []string{"ООО", "ИП", "ЗАО", "ОАО", "ПАО", "ТОО", "АО"}
	for _, keyword := range opfKeywords {
		if strings.Contains(name, keyword) {
			score += 10.0
			break
		}
	}

	// Quality score
	score += item.QualityScore * 20.0

	return score
}

// countFilledFields подсчитывает количество заполненных полей
func (cda *CounterpartyDuplicateAnalyzer) countFilledFields(item *CounterpartyDuplicateItem) int {
	count := 0
	if item.INN != "" {
		count++
	}
	if item.BIN != "" {
		count++
	}
	if item.KPP != "" {
		count++
	}
	if item.LegalAddress != "" {
		count++
	}
	if item.PostalAddress != "" {
		count++
	}
	if item.ContactPhone != "" {
		count++
	}
	if item.ContactEmail != "" {
		count++
	}
	if item.ContactPerson != "" {
		count++
	}
	if item.BankName != "" {
		count++
	}
	if item.BankAccount != "" {
		count++
	}
	if item.CorrespondentAccount != "" {
		count++
	}
	if item.BIK != "" {
		count++
	}
	return count
}

// getEnrichmentPriority возвращает приоритет источника обогащения
// Высший приоритет: Adata.kz (3.0), Dadata.ru (2.0), gisp.gov.ru (1.5), другие (1.0)
func (cda *CounterpartyDuplicateAnalyzer) getEnrichmentPriority(sourceEnrichment string) float64 {
	if sourceEnrichment == "" {
		return 0.0
	}

	sourceLower := strings.ToLower(sourceEnrichment)
	if strings.Contains(sourceLower, "adata.kz") || strings.Contains(sourceLower, "adata") {
		return 3.0
	}
	if strings.Contains(sourceLower, "dadata.ru") || strings.Contains(sourceLower, "dadata") {
		return 2.0
	}
	if strings.Contains(sourceLower, "gisp.gov.ru") || strings.Contains(sourceLower, "gisp") {
		return 1.5
	}
	return 1.0
}

// catalogItemToDuplicateItem преобразует CatalogItem в CounterpartyDuplicateItem с извлечением всех данных
func (cda *CounterpartyDuplicateAnalyzer) catalogItemToDuplicateItem(item *database.CatalogItem, inn, kpp, bin string) *CounterpartyDuplicateItem {
	duplicateItem := &CounterpartyDuplicateItem{
		ID:             item.ID,
		Reference:      item.Reference,
		Code:           item.Code,
		Name:           item.Name,
		Attributes:     item.Attributes, // Сохраняем оригинальные атрибуты
		INN:            inn,
		KPP:            kpp,
		BIN:            bin,
		QualityScore:   0.5,
		SourceDatabase: "",
	}

	// Извлекаем данные из атрибутов, если они не были переданы
	if item.Attributes != "" {
		if inn == "" {
			if extractedINN, err := extractors.ExtractINNFromAttributes(item.Attributes); err == nil {
				duplicateItem.INN = extractedINN
			}
		}
		if kpp == "" {
			if extractedKPP, err := extractors.ExtractKPPFromAttributes(item.Attributes); err == nil {
				duplicateItem.KPP = extractedKPP
			}
		}
		if bin == "" {
			if extractedBIN, err := extractors.ExtractBINFromAttributes(item.Attributes); err == nil {
				duplicateItem.BIN = extractedBIN
			}
		}

		// Извлекаем адреса
		if addr, err := extractors.ExtractAddressFromAttributes(item.Attributes); err == nil {
			duplicateItem.LegalAddress = addr
			duplicateItem.PostalAddress = addr
		}

		// Извлекаем контактную информацию
		if phone, err := extractors.ExtractContactPhoneFromAttributes(item.Attributes); err == nil {
			duplicateItem.ContactPhone = phone
		}
		if email, err := extractors.ExtractContactEmailFromAttributes(item.Attributes); err == nil {
			duplicateItem.ContactEmail = email
		}
		if person, err := extractors.ExtractContactPersonFromAttributes(item.Attributes); err == nil {
			duplicateItem.ContactPerson = person
		}

		// Извлекаем банковские реквизиты
		if bank, err := extractors.ExtractBankNameFromAttributes(item.Attributes); err == nil {
			duplicateItem.BankName = bank
		}
		if account, err := extractors.ExtractBankAccountFromAttributes(item.Attributes); err == nil {
			duplicateItem.BankAccount = account
		}
		if corrAccount, err := extractors.ExtractCorrespondentAccountFromAttributes(item.Attributes); err == nil {
			duplicateItem.CorrespondentAccount = corrAccount
		}
		if bik, err := extractors.ExtractBIKFromAttributes(item.Attributes); err == nil {
			duplicateItem.BIK = bik
		}
	}

	return duplicateItem
}

// FindDuplicatesForCounterparty находит дубликаты для конкретного контрагента
func (cda *CounterpartyDuplicateAnalyzer) FindDuplicatesForCounterparty(
	counterparty *database.CatalogItem,
	allCounterparties []*database.CatalogItem,
) []*CounterpartyDuplicateItem {
	duplicates := []*CounterpartyDuplicateItem{}

	// Извлекаем идентификаторы
	inn, _ := extractors.ExtractINNFromAttributes(counterparty.Attributes)
	kpp, _ := extractors.ExtractKPPFromAttributes(counterparty.Attributes)
	bin, _ := extractors.ExtractBINFromAttributes(counterparty.Attributes)

	// Ищем дубликаты по ИНН/КПП
	if inn != "" {
		for _, item := range allCounterparties {
			if item.ID == counterparty.ID {
				continue // Пропускаем сам элемент
			}

			itemINN, _ := extractors.ExtractINNFromAttributes(item.Attributes)
			itemKPP, _ := extractors.ExtractKPPFromAttributes(item.Attributes)

			// Проверяем совпадение по ИНН
			if itemINN == inn {
				// Если есть КПП, проверяем и его
				if kpp != "" && itemKPP != "" {
					if itemKPP == kpp {
						duplicateItem := cda.catalogItemToDuplicateItem(item, itemINN, itemKPP, "")
						duplicates = append(duplicates, duplicateItem)
					}
				} else {
					// Если КПП нет, считаем дубликатом по ИНН
					duplicateItem := cda.catalogItemToDuplicateItem(item, itemINN, "", "")
					duplicates = append(duplicates, duplicateItem)
				}
			}
		}
	}

	// Ищем дубликаты по БИН
	if bin != "" {
		for _, item := range allCounterparties {
			if item.ID == counterparty.ID {
				continue
			}

			itemBIN, err := extractors.ExtractBINFromAttributes(item.Attributes)
			if err == nil && itemBIN == bin {
				// Проверяем, не добавлен ли уже этот элемент
				alreadyAdded := false
				for _, dup := range duplicates {
					if dup.ID == item.ID {
						alreadyAdded = true
						break
					}
				}

				if !alreadyAdded {
					duplicateItem := cda.catalogItemToDuplicateItem(item, "", "", itemBIN)
					duplicates = append(duplicates, duplicateItem)
				}
			}
		}
	}

	return duplicates
}

// GetDuplicateSummary возвращает сводку по дубликатам
func (cda *CounterpartyDuplicateAnalyzer) GetDuplicateSummary(groups []CounterpartyDuplicateGroup) map[string]interface{} {
	totalGroups := len(groups)
	totalDuplicates := 0
	duplicatesByINNKPP := 0
	duplicatesByBIN := 0
	duplicatesByBoth := 0

	for _, group := range groups {
		totalDuplicates += len(group.Items)
		if strings.Contains(group.KeyType, "inn_kpp") && !strings.Contains(group.KeyType, "bin") {
			duplicatesByINNKPP++
		} else if strings.Contains(group.KeyType, "bin") && !strings.Contains(group.KeyType, "inn_kpp") {
			duplicatesByBIN++
		} else {
			duplicatesByBoth++
		}
	}

	return map[string]interface{}{
		"total_groups":          totalGroups,
		"total_duplicates":      totalDuplicates,
		"duplicates_by_inn_kpp": duplicatesByINNKPP,
		"duplicates_by_bin":     duplicatesByBIN,
		"duplicates_by_both":    duplicatesByBoth,
	}
}
