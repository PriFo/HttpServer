package extractors

import (
	"testing"
)

// TestExtractINNFromAttributes проверяет извлечение ИНН из XML атрибутов
func TestExtractINNFromAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantINN    string
		wantErr    bool
	}{
		{
			name:       "ИНН в тексте",
			attributes: "ИНН: 1234567890",
			wantINN:    "1234567890",
			wantErr:    false,
		},
		{
			name:       "ИНН 12 цифр",
			attributes: "ИНН: 123456789012",
			wantINN:    "123456789012",
			wantErr:    false,
		},
		{
			name:       "ИНН в нижнем регистре",
			attributes: "inn: 1234567890",
			wantINN:    "1234567890",
			wantErr:    false,
		},
		{
			name:       "ИНН без префикса",
			attributes: "1234567890",
			wantINN:    "1234567890",
			wantErr:    false,
		},
		{
			name:       "пустой XML",
			attributes: "",
			wantINN:    "",
			wantErr:    true,
		},
		{
			name:       "ИНН не найден",
			attributes: "какой-то текст без ИНН",
			wantINN:    "",
			wantErr:    true,
		},
		{
			name:       "ИНН в XML теге",
			attributes: "<ИНН>1234567890</ИНН>",
			wantINN:    "1234567890",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractINNFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractINNFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantINN {
				t.Errorf("ExtractINNFromAttributes() = %v, want %v", got, tt.wantINN)
			}
		})
	}
}

// TestExtractKPPFromAttributes проверяет извлечение КПП из XML атрибутов
func TestExtractKPPFromAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantKPP    string
		wantErr    bool
	}{
		{
			name:       "КПП в тексте",
			attributes: "КПП: 123456789",
			wantKPP:    "123456789",
			wantErr:    false,
		},
		{
			name:       "КПП в нижнем регистре",
			attributes: "kpp: 123456789",
			wantKPP:    "123456789",
			wantErr:    false,
		},
		{
			name:       "КПП без префикса",
			attributes: "123456789",
			wantKPP:    "123456789",
			wantErr:    false,
		},
		{
			name:       "пустой XML",
			attributes: "",
			wantKPP:    "",
			wantErr:    true,
		},
		{
			name:       "КПП не найден",
			attributes: "какой-то текст без КПП",
			wantKPP:    "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractKPPFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractKPPFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantKPP {
				t.Errorf("ExtractKPPFromAttributes() = %v, want %v", got, tt.wantKPP)
			}
		})
	}
}

// TestExtractBINFromAttributes проверяет извлечение БИН из XML атрибутов
func TestExtractBINFromAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantBIN    string
		wantErr    bool
	}{
		{
			name:       "БИН в тексте",
			attributes: "БИН: 123456789012",
			wantBIN:    "123456789012",
			wantErr:    false,
		},
		{
			name:       "БИН в нижнем регистре",
			attributes: "bin: 123456789012",
			wantBIN:    "123456789012",
			wantErr:    false,
		},
		{
			name:       "БИН без префикса",
			attributes: "123456789012",
			wantBIN:    "123456789012",
			wantErr:    false,
		},
		{
			name:       "пустой XML",
			attributes: "",
			wantBIN:    "",
			wantErr:    true,
		},
		{
			name:       "БИН не найден",
			attributes: "какой-то текст без БИН",
			wantBIN:    "",
			wantErr:    true,
		},
		{
			name:       "неправильная длина БИН",
			attributes: "1234567890",
			wantBIN:    "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBINFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractBINFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantBIN {
				t.Errorf("ExtractBINFromAttributes() = %v, want %v", got, tt.wantBIN)
			}
		})
	}
}

// TestExtractINNFromAttributes_EdgeCases проверяет граничные случаи для ИНН
func TestExtractINNFromAttributes_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantErr    bool
	}{
		{
			name:       "множественные числа",
			attributes: "ИНН: 1234567890, другой номер: 9876543210",
			wantErr:    false,
		},
		{
			name:       "ИНН с пробелами",
			attributes: "ИНН: 1234567890",
			wantErr:    false,
		},
		{
			name:       "только цифры без контекста",
			attributes: "123456789012345",
			wantErr:    false, // Найдет первое подходящее число
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractINNFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractINNFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == "" {
				t.Error("ExtractINNFromAttributes() returned empty string but no error")
			}
		})
	}
}

// TestExtractKPPFromAttributes_EdgeCases проверяет граничные случаи для КПП
func TestExtractKPPFromAttributes_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantErr    bool
	}{
		{
			name:       "КПП с пробелами",
			attributes: "КПП: 123456789",
			wantErr:    false,
		},
		{
			name:       "множественные 9-значные числа",
			attributes: "КПП: 123456789, другой: 987654321",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractKPPFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractKPPFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == "" {
				t.Error("ExtractKPPFromAttributes() returned empty string but no error")
			}
		})
	}
}

// TestExtractLegalFormFromAttributes проверяет извлечение организационно-правовой формы
func TestExtractLegalFormFromAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantForm   string
		wantErr    bool
	}{
		{
			name:       "ООО в тексте",
			attributes: "ООО Тестовая Компания",
			wantForm:   "ООО",
			wantErr:    false,
		},
		{
			name:       "ЗАО в тексте",
			attributes: "ЗАО Компания",
			wantForm:   "ЗАО",
			wantErr:    false,
		},
		{
			name:       "ИП в тексте",
			attributes: "ИП Иванов Иван Иванович",
			wantForm:   "ИП",
			wantErr:    false,
		},
		{
			name:       "ОПФ в XML теге",
			attributes: "<ОрганизационноПравоваяФорма>ООО</ОрганизационноПравоваяФорма>",
			wantForm:   "ООО",
			wantErr:    false,
		},
		{
			name:       "ОПФ с префиксом",
			attributes: "Организационно-правовая форма: ООО",
			wantForm:   "ООО",
			wantErr:    false,
		},
		{
			name:       "пустой XML",
			attributes: "",
			wantForm:   "",
			wantErr:    true,
		},
		{
			name:       "ОПФ не найдена",
			attributes: "какой-то текст без ОПФ",
			wantForm:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractLegalFormFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractLegalFormFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantForm {
				t.Errorf("ExtractLegalFormFromAttributes() = %v, want %v", got, tt.wantForm)
			}
		})
	}
}

// TestExtractBankNameFromAttributes проверяет извлечение названия банка
func TestExtractBankNameFromAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantBank   string
		wantErr    bool
	}{
		{
			name:       "Банк в тексте",
			attributes: "Банк: Сбербанк России",
			wantBank:   "Сбербанк России",
			wantErr:    false,
		},
		{
			name:       "Банк в XML теге",
			attributes: "<Банк>ВТБ</Банк>",
			wantBank:   "ВТБ",
			wantErr:    false,
		},
		{
			name:       "Банк с кавычками",
			attributes: `Банк: "Альфа-Банк"`,
			wantBank:   "Альфа-Банк",
			wantErr:    false,
		},
		{
			name:       "пустой XML",
			attributes: "",
			wantBank:   "",
			wantErr:    true,
		},
		{
			name:       "Банк не найден",
			attributes: "какой-то текст без банка",
			wantBank:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBankNameFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractBankNameFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantBank {
				t.Errorf("ExtractBankNameFromAttributes() = %v, want %v", got, tt.wantBank)
			}
		})
	}
}

// TestExtractBankAccountFromAttributes проверяет извлечение расчетного счета
func TestExtractBankAccountFromAttributes(t *testing.T) {
	tests := []struct {
		name        string
		attributes  string
		wantAccount string
		wantErr     bool
	}{
		{
			name:        "Расчетный счет в тексте",
			attributes:  "Расчетный счет: 40702810100000000001",
			wantAccount: "40702810100000000001",
			wantErr:     false,
		},
		{
			name:        "Р/С в тексте",
			attributes:  "Р/С: 40702810100000000001",
			wantAccount: "40702810100000000001",
			wantErr:     false,
		},
		{
			name:        "Счет в XML теге",
			attributes:  "<РасчетныйСчет>40702810100000000001</РасчетныйСчет>",
			wantAccount: "40702810100000000001",
			wantErr:     false,
		},
		{
			name:        "20-значное число",
			attributes:  "40702810100000000001",
			wantAccount: "40702810100000000001",
			wantErr:     false,
		},
		{
			name:        "пустой XML",
			attributes:  "",
			wantAccount: "",
			wantErr:     true,
		},
		{
			name:        "Счет не найден",
			attributes:  "какой-то текст без счета",
			wantAccount: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBankAccountFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractBankAccountFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantAccount {
				t.Errorf("ExtractBankAccountFromAttributes() = %v, want %v", got, tt.wantAccount)
			}
		})
	}
}

// TestExtractCorrespondentAccountFromAttributes проверяет извлечение корреспондентского счета
func TestExtractCorrespondentAccountFromAttributes(t *testing.T) {
	tests := []struct {
		name        string
		attributes  string
		wantAccount string
		wantErr     bool
	}{
		{
			name:        "Корреспондентский счет в тексте",
			attributes:  "Корреспондентский счет: 30101810100000000001",
			wantAccount: "30101810100000000001",
			wantErr:     false,
		},
		{
			name:        "К/С в тексте",
			attributes:  "К/С: 30101810100000000001",
			wantAccount: "30101810100000000001",
			wantErr:     false,
		},
		{
			name:        "Кор. счет в тексте",
			attributes:  "Кор. счет: 30101810100000000001",
			wantAccount: "30101810100000000001",
			wantErr:     false,
		},
		{
			name:        "пустой XML",
			attributes:  "",
			wantAccount: "",
			wantErr:     true,
		},
		{
			name:        "Корреспондентский счет не найден",
			attributes:  "какой-то текст без корреспондентского счета",
			wantAccount: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractCorrespondentAccountFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractCorrespondentAccountFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantAccount {
				t.Errorf("ExtractCorrespondentAccountFromAttributes() = %v, want %v", got, tt.wantAccount)
			}
		})
	}
}

// TestExtractBIKFromAttributes проверяет извлечение БИК банка
func TestExtractBIKFromAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantBIK    string
		wantErr    bool
	}{
		{
			name:       "БИК в тексте",
			attributes: "БИК: 044525225",
			wantBIK:    "044525225",
			wantErr:    false,
		},
		{
			name:       "БИК в нижнем регистре",
			attributes: "bik: 044525225",
			wantBIK:    "044525225",
			wantErr:    false,
		},
		{
			name:       "БИК в XML теге",
			attributes: "<БИК>044525225</БИК>",
			wantBIK:    "044525225",
			wantErr:    false,
		},
		{
			name:       "9-значное число",
			attributes: "044525225",
			wantBIK:    "044525225",
			wantErr:    false,
		},
		{
			name:       "пустой XML",
			attributes: "",
			wantBIK:    "",
			wantErr:    true,
		},
		{
			name:       "БИК не найден",
			attributes: "какой-то текст без БИК",
			wantBIK:    "",
			wantErr:    true,
		},
		{
			name:       "БИК не путается с КПП",
			attributes: "КПП: 123456789, БИК: 044525225",
			wantBIK:    "044525225",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBIKFromAttributes(tt.attributes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractBIKFromAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantBIK {
				t.Errorf("ExtractBIKFromAttributes() = %v, want %v", got, tt.wantBIK)
			}
		})
	}
}
