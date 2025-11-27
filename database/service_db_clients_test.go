package database

import "testing"

func newTestServiceDB(t *testing.T) *ServiceDB {
	t.Helper()

	db, err := NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create service DB: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	// В сервисной БД тестов нет таблицы uploads, но бизнес-логика ожидает её существование.
	// Создаем упрощённую версию, достаточную для запросов статистики.
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER,
			started_at TIMESTAMP,
			completed_at TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create uploads table: %v", err)
	}

	return db
}

func insertTestClientWithNulls(t *testing.T, db *ServiceDB) int64 {
	result, err := db.conn.Exec(`
		INSERT INTO clients (name, legal_name, description, contact_email, contact_phone, tax_id, country, status, created_by)
		VALUES (?, ?, NULL, NULL, NULL, NULL, NULL, NULL, NULL)
	`, "Клиент NULL", "ООО «Пусто»")
	if err != nil {
		t.Fatalf("failed to insert client: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get client id: %v", err)
	}

	return id
}

func TestServiceDB_GetAllClientsHandlesNulls(t *testing.T) {
	db := newTestServiceDB(t)
	insertTestClientWithNulls(t, db)

	clients, err := db.GetAllClients()
	if err != nil {
		t.Fatalf("GetAllClients returned error: %v", err)
	}

	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}

	client := clients[0]
	if client.Description != "" || client.ContactEmail != "" || client.ContactPhone != "" || client.TaxID != "" || client.Country != "" {
		t.Fatalf("expected empty strings for nullable fields, got %+v", client)
	}
}

func TestServiceDB_GetClientsByIDsHandlesNulls(t *testing.T) {
	db := newTestServiceDB(t)
	id := insertTestClientWithNulls(t, db)

	clients, err := db.GetClientsByIDs([]int{int(id)})
	if err != nil {
		t.Fatalf("GetClientsByIDs returned error: %v", err)
	}

	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}

	client := clients[0]
	if client.Description != "" || client.ContactEmail != "" || client.ContactPhone != "" || client.TaxID != "" || client.Country != "" {
		t.Fatalf("expected empty strings for nullable fields, got %+v", client)
	}
}

func TestServiceDB_GetClientsWithStatsHandlesNulls(t *testing.T) {
	db := newTestServiceDB(t)
	insertTestClientWithNulls(t, db)

	clients, err := db.GetClientsWithStats()
	if err != nil {
		t.Fatalf("GetClientsWithStats returned error: %v", err)
	}

	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}

	client := clients[0]
	if client["description"] != "" || client["country"] != "" || client["status"] != "" {
		t.Fatalf("expected empty strings for nullable fields, got %+v", client)
	}
}
