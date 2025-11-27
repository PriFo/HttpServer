package services

import (
	"testing"
)

// TestNewReclassificationService проверяет создание нового сервиса переклассификации
func TestNewReclassificationService(t *testing.T) {
	service := NewReclassificationService()
	if service == nil {
		t.Error("NewReclassificationService() should not return nil")
	}
}

// TestReclassificationService_Start проверяет запуск переклассификации
func TestReclassificationService_Start(t *testing.T) {
	service := NewReclassificationService()

	req := ReclassificationRequest{
		ClassifierID: 1,
		StrategyID:   "top_priority",
		Limit:        100,
	}

	err := service.Start(req)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !service.IsRunning() {
		t.Error("Expected service to be running after Start()")
	}
}

// TestReclassificationService_Start_AlreadyRunning проверяет обработку повторного запуска
func TestReclassificationService_Start_AlreadyRunning(t *testing.T) {
	service := NewReclassificationService()

	req := ReclassificationRequest{
		ClassifierID: 1,
		StrategyID:   "top_priority",
	}

	// Первый запуск
	err := service.Start(req)
	if err != nil {
		t.Fatalf("First Start() error = %v", err)
	}

	// Второй запуск должен вернуть ошибку
	err = service.Start(req)
	if err == nil {
		t.Error("Expected error when starting already running service")
	}
}

// TestReclassificationService_Start_DefaultValues проверяет значения по умолчанию
func TestReclassificationService_Start_DefaultValues(t *testing.T) {
	service := NewReclassificationService()

	req := ReclassificationRequest{
		ClassifierID: 0, // Должно быть установлено в 1
		StrategyID:   "", // Должно быть установлено в "top_priority"
	}

	err := service.Start(req)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

// TestReclassificationService_Stop проверяет остановку переклассификации
func TestReclassificationService_Stop(t *testing.T) {
	service := NewReclassificationService()

	req := ReclassificationRequest{
		ClassifierID: 1,
		StrategyID:   "top_priority",
	}

	// Запускаем
	err := service.Start(req)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Останавливаем
	service.Stop()

	if service.IsRunning() {
		t.Error("Expected service to not be running after Stop()")
	}
}

// TestReclassificationService_Stop_NotRunning проверяет остановку когда не запущена
func TestReclassificationService_Stop_NotRunning(t *testing.T) {
	service := NewReclassificationService()

	// Останавливаем без запуска
	service.Stop()

	if service.IsRunning() {
		t.Error("Expected service to not be running")
	}
}

// TestReclassificationService_GetStatus проверяет получение статуса
func TestReclassificationService_GetStatus(t *testing.T) {
	service := NewReclassificationService()

	status := service.GetStatus()
	if status.IsRunning {
		t.Error("Expected status to show not running initially")
	}
}

// TestReclassificationService_IsRunning проверяет проверку статуса запуска
func TestReclassificationService_IsRunning(t *testing.T) {
	service := NewReclassificationService()

	// Изначально не запущена
	if service.IsRunning() {
		t.Error("Expected service to not be running initially")
	}

	// Запускаем
	req := ReclassificationRequest{
		ClassifierID: 1,
		StrategyID:   "top_priority",
	}
	err := service.Start(req)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Теперь должна быть запущена
	if !service.IsRunning() {
		t.Error("Expected service to be running after Start()")
	}
}

