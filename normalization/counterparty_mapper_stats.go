package normalization

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// CounterpartyStatsReport описывает агрегированную статистику по контрагентам проекта.
type CounterpartyStatsReport struct {
	ProjectID                 int                     `json:"project_id"`
	TotalMappedCounterparties int                     `json:"total_mapped_counterparties"`
	UniqueNormalizedNames     int                     `json:"unique_normalized_names"`
	GroupsWithDuplicates      int                     `json:"groups_with_duplicates"`
	UnmatchedRecords          int                     `json:"unmatched_records"`
	TopGroups                 []DuplicateGroupSummary `json:"top_groups"`
	GeneratedAt               time.Time               `json:"generated_at"`
}

// DuplicateGroupSummary описывает одну группу дубликатов.
type DuplicateGroupSummary struct {
	Identifier string `json:"identifier"`
	KeyType    string `json:"key_type"`
	Count      int    `json:"count"`
}

// GetNormalizedCounterpartyStats возвращает аггрегированную статистику по проекту.
func (cm *CounterpartyMapper) GetNormalizedCounterpartyStats(projectID int) (*CounterpartyStatsReport, error) {
	if cm.serviceDB == nil {
		return nil, fmt.Errorf("serviceDB is nil")
	}

	if _, err := cm.serviceDB.GetClientProject(projectID); err != nil {
		return nil, fmt.Errorf("project %d not found: %w", projectID, err)
	}

	report := &CounterpartyStatsReport{
		ProjectID:   projectID,
		GeneratedAt: time.Now(),
	}

	var totalMapped int
	if err := cm.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM normalized_counterparties 
		WHERE client_project_id = ?`, projectID).Scan(&totalMapped); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count normalized counterparties: %w", err)
	}
	report.TotalMappedCounterparties = totalMapped

	var uniqueNames int
	if err := cm.serviceDB.QueryRow(`
		SELECT COUNT(DISTINCT normalized_name) 
		FROM normalized_counterparties 
		WHERE client_project_id = ? AND normalized_name IS NOT NULL AND normalized_name != ''`, projectID).Scan(&uniqueNames); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count unique normalized names: %w", err)
	}
	report.UniqueNormalizedNames = uniqueNames

	var unmatched int
	if err := cm.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM normalized_counterparties 
		WHERE client_project_id = ? 
		  AND COALESCE(tax_id, '') = '' 
		  AND COALESCE(bin, '') = ''`, projectID).Scan(&unmatched); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count unmatched records: %w", err)
	}
	report.UnmatchedRecords = unmatched

	groups := map[string]*DuplicateGroupSummary{}
	if err := cm.collectDuplicateGroups(projectID, groups); err != nil {
		return nil, err
	}

	report.GroupsWithDuplicates = len(groups)
	if len(groups) > 0 {
		var groupSlice []DuplicateGroupSummary
		for _, g := range groups {
			groupSlice = append(groupSlice, *g)
		}
		sort.Slice(groupSlice, func(i, j int) bool {
			if groupSlice[i].Count == groupSlice[j].Count {
				return groupSlice[i].Identifier < groupSlice[j].Identifier
			}
			return groupSlice[i].Count > groupSlice[j].Count
		})
		if len(groupSlice) > 5 {
			groupSlice = groupSlice[:5]
		}
		report.TopGroups = groupSlice
	}

	return report, nil
}

func (cm *CounterpartyMapper) collectDuplicateGroups(projectID int, groups map[string]*DuplicateGroupSummary) error {
	if err := cm.collectGroupedDuplicates(projectID, "tax_id", groups); err != nil {
		return err
	}
	if err := cm.collectGroupedDuplicates(projectID, "bin", groups); err != nil {
		return err
	}
	return nil
}

func (cm *CounterpartyMapper) collectGroupedDuplicates(projectID int, column string, groups map[string]*DuplicateGroupSummary) error {
	query := fmt.Sprintf(`
		SELECT %[1]s, COUNT(*) as cnt
		FROM normalized_counterparties
		WHERE client_project_id = ? AND %[1]s IS NOT NULL AND %[1]s != ''
		GROUP BY %[1]s
		HAVING cnt > 1
	`, column)

	rows, err := cm.serviceDB.Query(query, projectID)
	if err != nil {
		return fmt.Errorf("failed to query duplicate groups for %s: %w", column, err)
	}
	defer rows.Close()

	for rows.Next() {
		var identifier string
		var count int
		if err := rows.Scan(&identifier, &count); err != nil {
			return fmt.Errorf("failed to scan duplicate group for %s: %w", column, err)
		}
		key := column + ":" + identifier
		groups[key] = &DuplicateGroupSummary{
			Identifier: identifier,
			KeyType:    column,
			Count:      count,
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate duplicate groups for %s: %w", column, err)
	}

	return nil
}
