package database

import (
	"database/sql"
	"fmt"
	"time"
)

// ensureDemoClients инициализирует сервисную БД демо-данными, чтобы
// интерфейс сразу показывал хотя бы пару клиентов и проектов.
// Вызывается после создания схемы и выполняется только если таблица clients пуста.
func ensureDemoClients(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("service db connection is nil")
	}

	var clientCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM clients`).Scan(&clientCount); err != nil {
		return fmt.Errorf("failed to count clients: %w", err)
	}

	if clientCount > 0 {
		// Уже есть реальные данные — оставляем как есть.
		return nil
	}

	type demoProject struct {
		Name        string
		ProjectType string
		Description string
		Source      string
		TargetScore float64
	}

	type demoClient struct {
		Name        string
		LegalName   string
		Description string
		Email       string
		Phone       string
		TaxID       string
		Country     string
		Projects    []demoProject
	}

	demoData := []demoClient{
		{
			Name:        "АО «Цифровые решения»",
			LegalName:   "АО «Цифровые решения»",
			Description: "Крупный интегратор 1С, проекты по нормализации номенклатуры и контрагентов",
			Email:       "ops@digital-solutions.local",
			Phone:       "+7 (495) 123-45-67",
			TaxID:       "7701234567",
			Country:     "RU",
			Projects: []demoProject{
				{
					Name:        "Номенклатура ERP",
					ProjectType: "nomenclature",
					Description: "Нормализация каталога товаров ERP, >50k позиций",
					Source:      "1C:ERP",
					TargetScore: 0.92,
				},
				{
					Name:        "Контрагенты B2B",
					ProjectType: "counterparty",
					Description: "Очистка и нормализация контрагентов B2B сегмента",
					Source:      "1C:УПП",
					TargetScore: 0.9,
				},
			},
		},
		{
			Name:        "ТОО «Eurasia Trade»",
			LegalName:   "ТОО «Eurasia Trade»",
			Description: "Дистрибьютор промышленного оборудования. Ведёт проекты в Казахстане",
			Email:       "it@eurasiatrade.kz",
			Phone:       "+7 (727) 555-12-12",
			TaxID:       "601250789012",
			Country:     "KZ",
			Projects: []demoProject{
				{
					Name:        "Каталог ГМК",
					ProjectType: "nomenclature",
					Description: "Единый каталог для металлургического сегмента",
					Source:      "1C:UPP KZ",
					TargetScore: 0.9,
				},
				{
					Name:        "Контрагенты KZ",
					ProjectType: "counterparty",
					Description: "Нормализация казахстанских контрагентов с BIN",
					Source:      "1C:Бухгалтерия KZ",
					TargetScore: 0.88,
				},
			},
		},
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start demo seed transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	for _, client := range demoData {
		result, err := tx.Exec(
			`INSERT INTO clients (name, legal_name, description, contact_email, contact_phone, tax_id, country, status, created_by, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 'active', 'demo', ?, ?)`,
			client.Name,
			client.LegalName,
			client.Description,
			client.Email,
			client.Phone,
			client.TaxID,
			client.Country,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert demo client %q: %w", client.Name, err)
		}

		clientID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get client id for %q: %w", client.Name, err)
		}

		for _, project := range client.Projects {
			if _, err := tx.Exec(
				`INSERT INTO client_projects (client_id, name, project_type, description, source_system, target_quality_score, status, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
				clientID,
				project.Name,
				project.ProjectType,
				project.Description,
				project.Source,
				project.TargetScore,
				now,
				now,
			); err != nil {
				return fmt.Errorf("failed to insert demo project %q: %w", project.Name, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit demo client seed: %w", err)
	}

	return nil
}
