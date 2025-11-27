package extractors

import (
	"fmt"
	"regexp"
	"strings"
)

// ExtractINNFromAttributes извлекает ИНН из XML атрибутов
func ExtractINNFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	lowerXML := strings.ToLower(attributesXML)

	// Сначала пробуем найти ИНН в тексте с помощью регулярного выражения
	// Ищем паттерны типа "ИНН: 1234567890" или "inn: 1234567890"
	re := regexp.MustCompile(`(?i)(?:инн|inn)[\s:]*(\d{10,12})`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Пробуем парсить как XML с разными вариантами названий полей
	possibleFields := []string{"ИНН", "ИННКонтрагента", "ИННЮридическогоЛица", "inn", "INN", "TaxID", "TaxId", "tax_id"}

	// Если это похоже на XML, пробуем парсить
	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			// Пробуем найти поле в XML
			re := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field))
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				// Проверяем, что это число из 10 или 12 цифр
				if matched, _ := regexp.MatchString(`^\d{10,12}$`, value); matched {
					return value, nil
				}
			}
		}
	}

	// Пробуем найти ИНН как число из 10 цифр (но не в середине другого числа)
	re = regexp.MustCompile(`(?:^|[^\d])(\d{10})(?:[^\d]|$)`)
	matches = re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Если встретили длинную последовательность цифр, попробуем взять первые 10
	re = regexp.MustCompile(`\d{10,}`)
	longMatches := re.FindAllString(attributesXML, -1)
	for _, block := range longMatches {
		// Если рядом нет прямого упоминания БИН, считаем это ИНН
		if len(block) >= 10 && !(strings.Contains(lowerXML, "бин") || strings.Contains(lowerXML, "bin")) {
			return block[:10], nil
		}
	}

	return "", fmt.Errorf("ИНН not found in attributes")
}

// ExtractKPPFromAttributes извлекает КПП из XML атрибутов
func ExtractKPPFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Ищем КПП в тексте
	re := regexp.MustCompile(`(?i)(?:кпп|kpp)[\s:]*(\d{9})`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Пробуем найти КПП как число из 9 цифр
	re = regexp.MustCompile(`(\d{9})`)
	matches = re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("КПП not found in attributes")
}

// ExtractBINFromAttributes извлекает БИН (Бизнес-идентификационный номер) из XML атрибутов
// БИН используется в Казахстане и представляет собой 12-значный номер
func ExtractBINFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Сначала пробуем найти БИН в тексте с помощью регулярного выражения
	// Ищем паттерны типа "БИН: 123456789012" или "bin: 123456789012"
	re := regexp.MustCompile(`(?i)(?:бин|bin|бизнес[\s\-]*идентификационный[\s\-]*номер)[\s:]*(\d{12})`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Пробуем парсить как XML с разными вариантами названий полей
	possibleFields := []string{"БИН", "БИНКонтрагента", "БИНЮридическогоЛица", "bin", "BIN", "БизнесИдентификационныйНомер", "BINNumber"}

	// Если это похоже на XML, пробуем парсить
	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			// Пробуем найти поле в XML
			re := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field))
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				// Проверяем, что это число из 12 цифр
				if matched, _ := regexp.MatchString(`^\d{12}$`, value); matched {
					return value, nil
				}
			}
		}
	}

	// Пробуем найти БИН как число из 12 цифр (если это не ИНН)
	// Сначала проверяем, не является ли это ИНН (10 или 12 цифр)
	re = regexp.MustCompile(`(?:^|[^\d])(\d{12})(?:[^\d]|$)`)
	matches = re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		// Проверяем, не является ли это ИНН
		inn, _ := ExtractINNFromAttributes(attributesXML)
		if inn == "" || inn != matches[1] {
			// Если это не ИНН, то возможно это БИН
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("БИН not found in attributes")
}

// ExtractAddressFromAttributes извлекает адрес из XML атрибутов
func ExtractAddressFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Пробуем разные варианты названий полей для адреса
	possibleFields := []string{
		"Адрес", "АдресЮридический", "АдресПочтовый", "АдресФактический",
		"ЮридическийАдрес", "ПочтовыйАдрес", "ФактическийАдрес",
		"address", "legal_address", "postal_address", "actual_address",
	}

	// Ищем адрес по ключевым словам
	addressPatterns := []string{
		`(?i)(?:юридический\s*адрес|адрес\s*юридический)[\s:>]*([^<]+)`,
		`(?i)(?:почтовый\s*адрес|адрес\s*почтовый)[\s:>]*([^<]+)`,
		`(?i)(?:фактический\s*адрес|адрес\s*фактический)[\s:>]*([^<]+)`,
		`(?i)(?:адрес)[\s:>]*([^<]+)`,
	}

	for _, pattern := range addressPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			address := cleanAddressTail(matches[1])
			if len(address) > 10 { // Минимальная длина адреса
				return address, nil
			}
		}
	}

	// Пробуем найти адрес через XML парсинг
	for _, field := range possibleFields {
		pattern := fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field)
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			address := cleanAddressTail(matches[1])
			if len(address) > 10 {
				return address, nil
			}
		}
	}

	return "", fmt.Errorf("address not found in attributes")
}

func cleanAddressTail(address string) string {
	trimmed := strings.TrimSpace(address)
	lower := strings.ToLower(trimmed)
	cutMarkers := []string{
		", телефон", " телефон:", ",телефон", " тел.:", " tel", " phone",
		", email", " email", ", e-mail", " e-mail",
		", контакт", " контакт", ",контактное лицо", " контактное лицо",
		", банк", " банк:", ",банк", " р/с", " к/с", " бик", " inn", " бин",
	}
	cutPos := len(trimmed)
	for _, marker := range cutMarkers {
		if idx := strings.Index(lower, marker); idx != -1 && idx < cutPos {
			cutPos = idx
		}
	}
	result := strings.TrimSpace(trimmed[:cutPos])
	result = strings.TrimRight(result, ",;")
	return strings.TrimSpace(result)
}

// ExtractContactPhoneFromAttributes извлекает телефон из XML атрибутов
func ExtractContactPhoneFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Паттерны для поиска телефона
	phonePatterns := []string{
		`(?i)(?:телефон|phone|тел)[\s:]*([+]?[\d\s\-\(\)]{7,15})`,
		`(?i)(?:мобильный|mobile|сотовый)[\s:]*([+]?[\d\s\-\(\)]{7,15})`,
		`[+]?[\d\s\-\(\)]{10,15}`, // Простой паттерн для телефона
	}

	for _, pattern := range phonePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			phone := strings.TrimSpace(matches[1])
			// Очищаем от лишних символов, оставляем только цифры и +, -, (, )
			phone = regexp.MustCompile(`[^\d\+\-\(\)\s]`).ReplaceAllString(phone, "")
			if len(phone) >= 7 {
				return phone, nil
			}
		}
	}

	return "", fmt.Errorf("phone not found in attributes")
}

// ExtractContactEmailFromAttributes извлекает email из XML атрибутов
func ExtractContactEmailFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Паттерн для email
	emailPattern := `(?i)(?:email|e-mail|почта|электронная\s*почта)[\s:]*([a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})`
	re := regexp.MustCompile(emailPattern)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		return strings.TrimSpace(strings.ToLower(matches[1])), nil
	}

	// Пробуем найти email без префикса
	emailPattern2 := `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`
	re2 := regexp.MustCompile(emailPattern2)
	matches2 := re2.FindStringSubmatch(attributesXML)
	if len(matches2) > 0 {
		return strings.TrimSpace(strings.ToLower(matches2[0])), nil
	}

	return "", fmt.Errorf("email not found in attributes")
}

// ExtractContactPersonFromAttributes извлекает контактное лицо из XML атрибутов
func ExtractContactPersonFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Паттерны для поиска контактного лица
	personPatterns := []string{
		`(?i)(?:контактное\s*лицо|контактный|ответственное\s*лицо)[\s:]*([А-ЯЁа-яё\s]{5,50})`,
		`(?i)(?:директор|руководитель|менеджер)[\s:]*([А-ЯЁа-яё\s]{5,50})`,
	}

	for _, pattern := range personPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			person := strings.TrimSpace(matches[1])
			if len(person) >= 5 {
				return person, nil
			}
		}
	}

	return "", fmt.Errorf("contact person not found in attributes")
}

// ExtractLegalFormFromAttributes извлекает организационно-правовую форму из XML атрибутов
func ExtractLegalFormFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Паттерны для поиска организационно-правовой формы
	legalFormPatterns := []string{
		`(?i)(?:организационно[\s\-]*правовая\s*форма|опф|форма)[\s:]*([А-ЯЁа-яёА-ЯЁ\s]{2,20})`,
		`(?i)(?:ооо|общество\s*с\s*ограниченной\s*ответственностью)`,
		`(?i)(?:зао|закрытое\s*акционерное\s*общество)`,
		`(?i)(?:оао|открытое\s*акционерное\s*общество)`,
		`(?i)(?:ип|индивидуальный\s*предприниматель)`,
		`(?i)(?:пао|публичное\s*акционерное\s*общество)`,
		`(?i)(?:нп|некоммерческое\s*партнерство)`,
		`(?i)(?:ано|автономная\s*некоммерческая\s*организация)`,
		`(?i)(?:тоо|товарищество\s*с\s*ограниченной\s*ответственностью)`,
	}

	// Сначала ищем по ключевым словам
	for _, pattern := range legalFormPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			form := strings.TrimSpace(matches[1])
			if len(form) >= 2 {
				return form, nil
			}
		} else if re.MatchString(attributesXML) {
			// Если паттерн совпал, но без группы захвата, извлекаем аббревиатуру
			if strings.Contains(strings.ToLower(pattern), "ооо") {
				return "ООО", nil
			} else if strings.Contains(strings.ToLower(pattern), "зао") {
				return "ЗАО", nil
			} else if strings.Contains(strings.ToLower(pattern), "оао") {
				return "ОАО", nil
			} else if strings.Contains(strings.ToLower(pattern), "ип") {
				return "ИП", nil
			} else if strings.Contains(strings.ToLower(pattern), "пао") {
				return "ПАО", nil
			} else if strings.Contains(strings.ToLower(pattern), "тоо") {
				return "ТОО", nil
			}
		}
	}

	// Пробуем найти через XML парсинг
	possibleFields := []string{
		"ОрганизационноПравоваяФорма", "ОПФ", "Форма", "LegalForm", "legal_form",
		"ОрганизационнаяФорма", "ОргФорма",
	}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			pattern := fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field)
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				form := strings.TrimSpace(matches[1])
				if len(form) >= 2 {
					return form, nil
				}
			}
		}
	}

	return "", fmt.Errorf("legal form not found in attributes")
}

// ExtractBankNameFromAttributes извлекает название банка из XML атрибутов
func ExtractBankNameFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Паттерны для поиска названия банка
	bankPatterns := []string{
		`(?i)(?:банк|bank)[\s:]*([А-ЯЁа-яё\s"«»\-]{2,100})`,
		`(?i)(?:наименование\s*банка|банк\s*получателя)[\s:]*([А-ЯЁа-яё\s"«»\-]{2,100})`,
	}

	for _, pattern := range bankPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			bank := strings.TrimSpace(matches[1])
			// Очищаем от кавычек и лишних символов
			bank = strings.Trim(bank, `"«»'`)
			if len(bank) >= 3 {
				return bank, nil
			}
		}
	}

	// Пробуем найти через XML парсинг
	possibleFields := []string{
		"Банк", "БанкПолучателя", "НаименованиеБанка", "BankName", "bank_name",
		"БанкПлательщика", "БанкКонтрагента",
	}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			pattern := fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field)
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				bank := strings.TrimSpace(matches[1])
				bank = strings.Trim(bank, `"«»'`)
				if len(bank) >= 3 {
					return bank, nil
				}
			}
		}
	}

	return "", fmt.Errorf("bank name not found in attributes")
}

// ExtractBankAccountFromAttributes извлекает расчетный счет из XML атрибутов
func ExtractBankAccountFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Паттерны для поиска расчетного счета (обычно 20 цифр)
	accountPatterns := []string{
		`(?i)(?:расчетный\s*счет|р\/с|рс|счет)[\s:]*(\d{20})`,
		`(?i)(?:счет\s*получателя|счет\s*плательщика)[\s:]*(\d{20})`,
		`(?i)(?:банковский\s*счет)[\s:]*(\d{20})`,
	}

	for _, pattern := range accountPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			account := strings.TrimSpace(matches[1])
			if len(account) == 20 {
				return account, nil
			}
		}
	}

	// Пробуем найти 20-значное число (расчетный счет)
	re := regexp.MustCompile(`(?:^|[^\d])(\d{20})(?:[^\d]|$)`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// Пробуем найти через XML парсинг
	possibleFields := []string{
		"РасчетныйСчет", "РС", "Счет", "BankAccount", "bank_account",
		"СчетПолучателя", "СчетПлательщика", "БанковскийСчет",
	}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			pattern := fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field)
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				account := strings.TrimSpace(matches[1])
				// Удаляем пробелы и проверяем длину
				account = regexp.MustCompile(`\s`).ReplaceAllString(account, "")
				if matched, _ := regexp.MatchString(`^\d{20}$`, account); matched {
					return account, nil
				}
			}
		}
	}

	return "", fmt.Errorf("bank account not found in attributes")
}

// ExtractCorrespondentAccountFromAttributes извлекает корреспондентский счет из XML атрибутов
func ExtractCorrespondentAccountFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Паттерны для поиска корреспондентского счета (обычно 20 цифр)
	corrAccountPatterns := []string{
		`(?i)(?:корреспондентский\s*счет|к\/с|кс)[\s:]*(\d{20})`,
		`(?i)(?:кор\.\s*счет)[\s:]*(\d{20})`,
	}

	for _, pattern := range corrAccountPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			account := strings.TrimSpace(matches[1])
			if len(account) == 20 {
				return account, nil
			}
		}
	}

	// Пробуем найти через XML парсинг
	possibleFields := []string{
		"КорреспондентскийСчет", "КС", "КорСчет", "CorrespondentAccount", "correspondent_account",
	}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			pattern := fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field)
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				account := strings.TrimSpace(matches[1])
				// Удаляем пробелы и проверяем длину
				account = regexp.MustCompile(`\s`).ReplaceAllString(account, "")
				if matched, _ := regexp.MatchString(`^\d{20}$`, account); matched {
					return account, nil
				}
			}
		}
	}

	return "", fmt.Errorf("correspondent account not found in attributes")
}

// ExtractBIKFromAttributes извлекает БИК банка из XML атрибутов
func ExtractBIKFromAttributes(attributesXML string) (string, error) {
	if attributesXML == "" {
		return "", fmt.Errorf("empty attributes XML")
	}

	// Паттерны для поиска БИК (9 цифр)
	bikPatterns := []string{
		`(?i)(?:бик|bik)[\s:]*(\d{9})`,
		`(?i)(?:банковский\s*идентификационный\s*код)[\s:]*(\d{9})`,
	}

	for _, pattern := range bikPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(attributesXML)
		if len(matches) > 1 {
			bik := strings.TrimSpace(matches[1])
			if len(bik) == 9 {
				return bik, nil
			}
		}
	}

	// Пробуем найти 9-значное число (БИК)
	re := regexp.MustCompile(`(?:^|[^\d])(\d{9})(?:[^\d]|$)`)
	allMatches := re.FindAllStringSubmatchIndex(attributesXML, -1)
	lower := strings.ToLower(attributesXML)
	for _, idxs := range allMatches {
		if len(idxs) < 4 {
			continue
		}
		groupStart := idxs[2]
		groupEnd := idxs[3]
		candidate := strings.TrimSpace(attributesXML[groupStart:groupEnd])

		// Смотрим на несколько символов перед числом, чтобы не перепутать с КПП
		prefixStart := groupStart - 16
		if prefixStart < 0 {
			prefixStart = 0
		}
		prefix := lower[prefixStart:groupStart]
		if strings.Contains(prefix, "кпп") || strings.Contains(prefix, "kpp") {
			continue
		}

		if len(candidate) == 9 {
			return candidate, nil
		}
	}

	// Пробуем найти через XML парсинг
	possibleFields := []string{
		"БИК", "БанковскийИдентификационныйКод", "BIK", "bik",
	}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			pattern := fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field)
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				bik := strings.TrimSpace(matches[1])
				// Удаляем пробелы и проверяем длину
				bik = regexp.MustCompile(`\s`).ReplaceAllString(bik, "")
				if matched, _ := regexp.MatchString(`^\d{9}$`, bik); matched {
					return bik, nil
				}
			}
		}
	}

	return "", fmt.Errorf("BIK not found in attributes")
}
