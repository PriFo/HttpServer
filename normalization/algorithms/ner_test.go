package algorithms

import (
	"testing"
)

func TestRussianNER_ExtractEntities(t *testing.T) {
	ner := NewRussianNER()

	tests := []struct {
		name     string
		text     string
		expected int // Минимальное количество сущностей
		types    []NEREntityType
	}{
		{
			name:     "Материал и цвет",
			text:     "стальной белый кабель",
			expected: 2,
			types:    []NEREntityType{NEREntityTypeMaterial, NEREntityTypeColor},
		},
		{
			name:     "Размер и материал",
			text:     "деревянный шкаф 100x200",
			expected: 2,
			types:    []NEREntityType{NEREntityTypeMaterial, NEREntityTypeDimension},
		},
		{
			name:     "Вес и длина",
			text:     "кабель 2.5мм 100м 5кг",
			expected: 3,
			types:    []NEREntityType{NEREntityTypeLength, NEREntityTypeLength, NEREntityTypeWeight},
		},
		{
			name:     "Тип кабеля",
			text:     "многожильный медный кабель",
			expected: 2,
			types:    []NEREntityType{NEREntityTypeType, NEREntityTypeMaterial},
		},
		{
			name:     "Код",
			text:     "ER-00013004 кабель",
			expected: 1,
			types:    []NEREntityType{NEREntityTypeCode},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := ner.ExtractEntities(tt.text)
			
			if len(entities) < tt.expected {
				t.Errorf("ExtractEntities() found %d entities, want at least %d", len(entities), tt.expected)
			}
			
			// Проверяем типы
			foundTypes := make(map[NEREntityType]bool)
			for _, entity := range entities {
				foundTypes[entity.Type] = true
			}
			
			for _, expectedType := range tt.types {
				if !foundTypes[expectedType] {
					t.Errorf("Expected entity type %s not found", expectedType)
				}
			}
		})
	}
}

func TestRussianNER_ExtractMaterials(t *testing.T) {
	ner := NewRussianNER()

	text := "стальной деревянный пластиковый кабель"
	entities := ner.extractMaterials(text)

	if len(entities) < 3 {
		t.Errorf("extractMaterials() found %d materials, want at least 3", len(entities))
	}

	// Проверяем, что все материалы найдены
	materials := make(map[string]bool)
	for _, entity := range entities {
		if entity.Type == NEREntityTypeMaterial {
			materials[entity.Value] = true
		}
	}

	expectedMaterials := []string{"сталь", "дерево", "пластик"}
	for _, expected := range expectedMaterials {
		if !materials[expected] {
			t.Errorf("Expected material %s not found", expected)
		}
	}
}

func TestRussianNER_ExtractColors(t *testing.T) {
	ner := NewRussianNER()

	text := "белый черный красный кабель"
	entities := ner.extractColors(text)

	if len(entities) < 3 {
		t.Errorf("extractColors() found %d colors, want at least 3", len(entities))
	}

	colors := make(map[string]bool)
	for _, entity := range entities {
		if entity.Type == NEREntityTypeColor {
			colors[entity.Value] = true
		}
	}

	expectedColors := []string{"белый", "черный", "красный"}
	for _, expected := range expectedColors {
		if !colors[expected] {
			t.Errorf("Expected color %s not found", expected)
		}
	}
}

func TestRussianNER_ExtractDimensions(t *testing.T) {
	ner := NewRussianNER()

	text := "шкаф 100x200 50x30"
	entities := ner.extractDimensions(text)

	if len(entities) < 2 {
		t.Errorf("extractDimensions() found %d dimensions, want at least 2", len(entities))
	}

	for _, entity := range entities {
		if entity.Type != NEREntityTypeDimension {
			t.Errorf("Expected entity type DIMENSION, got %s", entity.Type)
		}
		if entity.Confidence != 1.0 {
			t.Errorf("Expected confidence 1.0, got %f", entity.Confidence)
		}
	}
}

func TestRussianNER_TagWithBIO(t *testing.T) {
	ner := NewRussianNER()

	text := "стальной белый кабель 100x200"
	tokens := ner.TagWithBIO(text)

	if len(tokens) == 0 {
		t.Error("TagWithBIO() returned no tokens")
	}

	// Проверяем, что есть хотя бы одна сущность с тегом B или I
	foundEntity := false
	for _, token := range tokens {
		if token.Tag == BIO_B || token.Tag == BIO_I {
			foundEntity = true
			break
		}
	}

	if !foundEntity {
		t.Error("No entities found with BIO tags")
	}
}

func TestRussianNER_AddMaterial(t *testing.T) {
	ner := NewRussianNER()

	// Добавляем слово "титановый" в словарь
	ner.AddMaterial("титановый", "титан")
	
	text := "титановый кабель"
	entities := ner.ExtractEntities(text)

	found := false
	for _, entity := range entities {
		if entity.Type == NEREntityTypeMaterial && (entity.Value == "титан" || entity.Text == "титановый") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Added material not found")
	}
}

