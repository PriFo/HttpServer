package classification

import "errors"

var (
	// ErrInvalidEntityID ошибка при невалидном ID сущности
	ErrInvalidEntityID = errors.New("invalid entity ID")
	
	// ErrInvalidCategory ошибка при невалидной категории
	ErrInvalidCategory = errors.New("invalid category")
	
	// ErrClassificationNotFound ошибка при отсутствии классификации
	ErrClassificationNotFound = errors.New("classification not found")
	
	// ErrClassificationAlreadyExists ошибка при попытке создать дублирующую классификацию
	ErrClassificationAlreadyExists = errors.New("classification already exists")
	
	// ErrInvalidConfidence ошибка при невалидном уровне уверенности
	ErrInvalidConfidence = errors.New("invalid confidence value")
)

