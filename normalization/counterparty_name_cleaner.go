package normalization

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CounterpartyNameNormalizationSummary содержит результат пакетной нормализации названий.
type CounterpartyNameNormalizationSummary struct {
	ProjectID             int           `json:"project_id"`
	TotalRecords          int           `json:"total_records"`
	UpdatedRecords        int           `json:"updated_records"`
	UpdatedNameCount      int           `json:"updated_name_count"`
	UpdatedLegalFormCount int           `json:"updated_legal_form_count"`
	SkippedWithoutName    int           `json:"skipped_without_name"`
	AppliedUpdates        int           `json:"applied_updates"`
	DryRun                bool          `json:"dry_run"`
	Duration              time.Duration `json:"duration"`
}

// NormalizeNamesForProject удаляет ОПФ из названий и нормализует legal_form.
func (cm *CounterpartyMapper) NormalizeNamesForProject(projectID int, dryRun bool) (*CounterpartyNameNormalizationSummary, error) {
	if cm.serviceDB == nil {
		return nil, fmt.Errorf("serviceDB is nil")
	}

	if _, err := cm.serviceDB.GetClientProject(projectID); err != nil {
		return nil, fmt.Errorf("project %d not found: %w", projectID, err)
	}

	cm.logger.Info("Starting counterparty name normalization",
		"project_id", projectID,
		"dry_run", dryRun)

	rows, err := cm.serviceDB.Query(`
		SELECT id, COALESCE(source_name, ''), COALESCE(normalized_name, ''), COALESCE(legal_form, '')
		FROM normalized_counterparties
		WHERE client_project_id = ?
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch normalized counterparties: %w", err)
	}
	defer rows.Close()

	summary := &CounterpartyNameNormalizationSummary{
		ProjectID: projectID,
		DryRun:    dryRun,
	}
	start := time.Now()

	var executor execRunner = cm.serviceDB.GetDB()
	var tx *sql.Tx
	if !dryRun {
		tx, err = cm.serviceDB.GetDB().Begin()
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		executor = tx
	}

	updateStmt := `
		UPDATE normalized_counterparties
		SET normalized_name = ?, legal_form = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	for rows.Next() {
		var (
			id             int
			sourceName     string
			normalizedName string
			legalForm      string
		)

		if err := rows.Scan(&id, &sourceName, &normalizedName, &legalForm); err != nil {
			if tx != nil {
				_ = tx.Rollback()
			}
			return nil, fmt.Errorf("failed to scan counterparty: %w", err)
		}

		summary.TotalRecords++

		rawName := normalizedName
		if strings.TrimSpace(rawName) == "" {
			rawName = sourceName
		}

		cleanName, canonicalForm := normalizeCounterpartyNameAndForm(rawName, legalForm)
		if cleanName == "" {
			summary.SkippedWithoutName++
			continue
		}

		updatedName := cleanName != normalizedName
		updatedForm := canonicalForm != strings.TrimSpace(legalForm)

		if updatedName {
			summary.UpdatedNameCount++
		}
		if updatedForm {
			summary.UpdatedLegalFormCount++
		}

		if updatedName || updatedForm {
			summary.UpdatedRecords++
			if !dryRun {
				var legalFormValue interface{}
				if canonicalForm != "" {
					legalFormValue = canonicalForm
				}
				if _, err := executor.Exec(updateStmt, cleanName, legalFormValue, id); err != nil {
					_ = tx.Rollback()
					return nil, fmt.Errorf("failed to update counterparty %d: %w", id, err)
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		if tx != nil {
			_ = tx.Rollback()
		}
		return nil, fmt.Errorf("failed to iterate counterparties: %w", err)
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit normalization changes: %w", err)
		}
		summary.AppliedUpdates = summary.UpdatedRecords
	}

	summary.Duration = time.Since(start)

	cm.logger.Info("Finished counterparty name normalization",
		"project_id", projectID,
		"total", summary.TotalRecords,
		"updated_records", summary.UpdatedRecords,
		"updated_names", summary.UpdatedNameCount,
		"updated_legal_forms", summary.UpdatedLegalFormCount,
		"skipped", summary.SkippedWithoutName,
		"dry_run", dryRun,
		"duration", summary.Duration)

	return summary, nil
}

type execRunner interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// normalizeCounterpartyNameAndForm приводит название к чистому виду и определяет ОПФ.
func normalizeCounterpartyNameAndForm(name, existingForm string) (string, string) {
	cleanName := cleanupCounterpartyName(name)
	inferredForm, strippedName := extractLegalFormFromName(cleanName)

	canonicalForm := normalizeLegalFormValue(existingForm)
	if canonicalForm == "" {
		canonicalForm = normalizeLegalFormValue(inferredForm)
	}

	if strippedName == "" {
		strippedName = cleanName
	}

	return strippedName, canonicalForm
}

func cleanupCounterpartyName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Trim(trimmed, `"'«»“”`)
	trimmed = whitespaceRegex.ReplaceAllString(trimmed, " ")
	return strings.TrimSpace(trimmed)
}

func extractLegalFormFromName(name string) (string, string) {
	for _, pattern := range legalFormPrefixPatterns {
		if matches := pattern.regex.FindStringSubmatch(name); len(matches) == 2 {
			return pattern.canonical, cleanupCounterpartyName(matches[1])
		}
	}

	for _, pattern := range legalFormSuffixPatterns {
		if matches := pattern.regex.FindStringSubmatch(name); len(matches) == 2 {
			return pattern.canonical, cleanupCounterpartyName(matches[1])
		}
	}

	return "", cleanupCounterpartyName(name)
}

func normalizeLegalFormValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	key := sanitizeLegalFormToken(value)
	if canonical, ok := legalFormAlias[key]; ok {
		return canonical
	}
	return strings.TrimSpace(value)
}

func sanitizeLegalFormToken(value string) string {
	replacer := strings.NewReplacer(
		" ", "",
		".", "",
		",", "",
		`"`, "",
		"«", "",
		"»", "",
		"“", "",
		"”", "",
		"'", "",
		"-", "",
		"_", "",
	)
	return replacer.Replace(strings.ToUpper(value))
}

var (
	whitespaceRegex         = regexp.MustCompile(`\s+`)
	legalFormPrefixPatterns []legalFormPattern
	legalFormSuffixPatterns []legalFormPattern
	legalFormAlias          = map[string]string{
		"ООО": "ООО",
		"ОБЩЕСТВОСОГРАНИЧЕННОЙОТВЕТСТВЕННОСТЬЮ": "ООО",
		"ЗАО": "ЗАО",
		"ЗАКРЫТОЕАКЦИОНЕРНОЕОБЩЕСТВО": "ЗАО",
		"ОАО": "ОАО",
		"ОТКРЫТОЕАКЦИОНЕРНОЕОБЩЕСТВО": "ОАО",
		"ПАО": "ПАО",
		"ПУБЛИЧНОЕАКЦИОНЕРНОЕОБЩЕСТВО": "ПАО",
		"АО":                  "АО",
		"АКЦИОНЕРНОЕОБЩЕСТВО": "АО",
		"НАО":                 "НАО",
		"НЕКОММЕРЧЕСКОЕАКЦИОНЕРНОЕОБЩЕСТВО": "НАО",
		"АОЗТ": "АОЗТ",
		"ИП":   "ИП",
		"ИНДИВИДУАЛЬНЫЙПРЕДПРИНИМАТЕЛЬ": "ИП",
		"ТОО": "ТОО",
		"ТОВАРИЩЕСТВОСОГРАНИЧЕННОЙОТВЕТСТВЕННОСТЬЮ": "ТОО",
		"ЧП":                          "ЧП",
		"ЧАСТНОЕПРЕДПРИЯТИЕ":          "ЧП",
		"LLC":                         "LLC",
		"LIMITEDLIABILITYCOMPANY":     "LLC",
		"LLP":                         "LLP",
		"LIMITEDLIABILITYPARTNERSHIP": "LLP",
		"JSC":                         "JSC",
		"JOINTSTOCKCOMPANY":           "JSC",
	}
	legalFormSynonyms = map[string][]string{
		"ООО": {
			`ООО`,
			`О\.?О\.?О\.?`,
			`Общество\s+с\s+ограниченной\s+ответственностью`,
		},
		"ЗАО": {
			`ЗАО`,
			`Закрытое\s+акционерное\s+общество`,
		},
		"ОАО": {
			`ОАО`,
			`Открытое\s+акционерное\s+общество`,
		},
		"ПАО": {
			`ПАО`,
			`Публичное\s+акционерное\s+общество`,
		},
		"АО": {
			`АО`,
			`Акционерное\s+общество`,
		},
		"НАО": {
			`НАО`,
			`Некоммерческое\s+акционерное\s+общество`,
		},
		"АОЗТ": {
			`АОЗТ`,
			`Акционерное\s+общество\s+закрытого\s+типа`,
		},
		"ИП": {
			`ИП`,
			`Индивидуальный\s+предприниматель`,
		},
		"ТОО": {
			`ТОО`,
			`Товарищество\s+с\s+ограниченной\s+ответственностью`,
		},
		"ЧП": {
			`ЧП`,
			`Частное\s+предприятие`,
		},
		"LLC": {
			`LLC`,
			`Limited\s+Liability\s+Company`,
		},
		"LLP": {
			`LLP`,
			`Limited\s+Liability\s+Partnership`,
		},
		"JSC": {
			`JSC`,
			`Joint\s+Stock\s+Company`,
		},
	}
)

type legalFormPattern struct {
	canonical string
	regex     *regexp.Regexp
}

func init() {
	for canonical, patterns := range legalFormSynonyms {
		alternation := strings.Join(patterns, "|")
		prefix := regexp.MustCompile(fmt.Sprintf(`^(?i)\s*(?:%s)\s*[«"“”']?(.*)$`, alternation))
		suffix := regexp.MustCompile(fmt.Sprintf(`^(?i)\s*[«"“”']?(.+?)[«"“”']?\s*(?:%s)\.?$`, alternation))
		legalFormPrefixPatterns = append(legalFormPrefixPatterns, legalFormPattern{
			canonical: canonical,
			regex:     prefix,
		})
		legalFormSuffixPatterns = append(legalFormSuffixPatterns, legalFormPattern{
			canonical: canonical,
			regex:     suffix,
		})
	}
}
