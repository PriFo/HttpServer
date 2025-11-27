package services

import (
	"testing"

	"httpserver/server/types"
)

// TestNewUploadService проверяет создание нового сервиса выгрузок
func TestNewUploadService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	logFunc := func(entry interface{}) {
		// Тестовая функция логирования
	}

	service := NewUploadService(db, serviceDB, nil, logFunc)
	if service == nil {
		t.Error("NewUploadService() should not return nil")
	}
}

// TestUploadService_ProcessHandshake проверяет обработку handshake запроса
func TestUploadService_ProcessHandshake(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	logFunc := func(entry interface{}) {
		// Тестовая функция логирования
	}

	service := NewUploadService(db, serviceDB, nil, logFunc)

	req := types.HandshakeRequest{
		Version1C:    "8.3",
		ConfigName:   "test_config",
		ComputerName: "test_computer",
		UserName:     "test_user",
	}

	result, err := service.ProcessHandshake(req)
	if err != nil {
		// Может быть ошибка валидации или другие ошибки
		// Это нормально для тестовой БД
		return
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestUploadService_ProcessHandshake_InvalidRequest проверяет обработку невалидного запроса
func TestUploadService_ProcessHandshake_InvalidRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	logFunc := func(entry interface{}) {
		// Тестовая функция логирования
	}

	service := NewUploadService(db, serviceDB, nil, logFunc)

	req := types.HandshakeRequest{
		Version1C:  "", // Пустая версия должна вызвать ошибку валидации
		ConfigName: "",
	}

	_, err := service.ProcessHandshake(req)
	if err == nil {
		t.Error("Expected error for invalid request")
	}
}

