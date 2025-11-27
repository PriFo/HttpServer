package quality

import (
	"testing"
)

func TestValidateINN(t *testing.T) {
	tests := []struct {
		name    string
		inn     string
		want    bool
	}{
		{
			name: "valid 10-digit INN",
			inn:  "7707083893",
			want: true,
		},
		{
			name: "valid 12-digit INN",
			inn:  "500100732259",
			want: true,
		},
		{
			name: "invalid length 9 digits",
			inn:  "123456789",
			want: false,
		},
		{
			name: "invalid length 11 digits",
			inn:  "12345678901",
			want: false,
		},
		{
			name: "invalid with letters",
			inn:  "770708389a",
			want: false,
		},
		{
			name: "valid with spaces cleaned",
			inn:  "7707 083 893",
			want: true,
		},
		{
			name: "valid with dashes cleaned",
			inn:  "7707-083-893",
			want: true,
		},
		{
			name: "empty string",
			inn:  "",
			want: false,
		},
		{
			name: "invalid checksum 10-digit",
			inn:  "1234567890",
			want: false,
		},
		{
			name: "invalid checksum 12-digit",
			inn:  "123456789012",
			want: false,
		},
		{
			name: "valid with leading/trailing spaces",
			inn:  " 7707083893 ",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateINN(tt.inn)
			if got != tt.want {
				t.Errorf("ValidateINN(%q) = %v, want %v", tt.inn, got, tt.want)
			}
		})
	}
}

func TestValidateKPP(t *testing.T) {
	tests := []struct {
		name string
		kpp  string
		want bool
	}{
		{
			name: "valid KPP",
			kpp:  "770701001",
			want: true,
		},
		{
			name: "invalid length 8 digits",
			kpp:  "77070100",
			want: false,
		},
		{
			name: "invalid length 10 digits",
			kpp:  "7707010010",
			want: false,
		},
		{
			name: "invalid with letters",
			kpp:  "77070100a",
			want: false,
		},
		{
			name: "valid with spaces cleaned",
			kpp:  "7707 010 01",
			want: true,
		},
		{
			name: "valid with dashes cleaned",
			kpp:  "7707-010-01",
			want: true,
		},
		{
			name: "empty string",
			kpp:  "",
			want: false,
		},
		{
			name: "valid with spaces cleaned",
			kpp:  " 770701001 ",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateKPP(tt.kpp)
			if got != tt.want {
				t.Errorf("ValidateKPP(%q) = %v, want %v", tt.kpp, got, tt.want)
			}
		})
	}
}

func TestExtractINNFromAttributes(t *testing.T) {
	tests := []struct {
		name        string
		attributes  string
		wantINN     string
		wantError   bool
	}{
		{
			name:       "INN in XML tag",
			attributes: "<ИНН>7707083893</ИНН>",
			wantINN:    "7707083893",
			wantError:  false,
		},
		{
			name:       "INN with label",
			attributes: "ИНН: 7707083893",
			wantINN:    "7707083893",
			wantError:  false,
		},
		{
			name:       "INN with label lowercase",
			attributes: "инн: 500100732259",
			wantINN:    "500100732259",
			wantError:  false,
		},
		{
			name:       "INN 12 digits",
			attributes: "ИНН: 500100732259",
			wantINN:    "500100732259",
			wantError:  false,
		},
		{
			name:       "INN as standalone number",
			attributes: "Some text 7707083893 more text",
			wantINN:    "7707083893",
			wantError:  false,
		},
		{
			name:       "empty attributes",
			attributes: "",
			wantINN:    "",
			wantError:  true,
		},
		{
			name:       "no INN found",
			attributes: "Some text without INN",
			wantINN:    "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractINNFromAttributes(tt.attributes)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractINNFromAttributes() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantINN {
				t.Errorf("ExtractINNFromAttributes() = %v, want %v", got, tt.wantINN)
			}
		})
	}
}

func TestExtractKPPFromAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes string
		wantKPP    string
		wantError  bool
	}{
		{
			name:       "KPP with label",
			attributes: "КПП: 770701001",
			wantKPP:    "770701001",
			wantError:  false,
		},
		{
			name:       "KPP with label lowercase",
			attributes: "кпп: 770701001",
			wantKPP:    "770701001",
			wantError:  false,
		},
		{
			name:       "KPP as standalone number",
			attributes: "Some text 770701001 more text",
			wantKPP:    "770701001",
			wantError:  false,
		},
		{
			name:       "empty attributes",
			attributes: "",
			wantKPP:    "",
			wantError:  true,
		},
		{
			name:       "no KPP found",
			attributes: "Some text without KPP",
			wantKPP:    "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractKPPFromAttributes(tt.attributes)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractKPPFromAttributes() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantKPP {
				t.Errorf("ExtractKPPFromAttributes() = %v, want %v", got, tt.wantKPP)
			}
		})
	}
}

func TestExtractAddressFromAttributes(t *testing.T) {
	tests := []struct {
		name        string
		attributes  string
		wantAddress string
		wantError   bool
	}{
		{
			name:        "legal address with label",
			attributes:  "Юридический адрес: г. Москва, ул. Ленина, д. 1",
			wantAddress: "г. Москва, ул. Ленина, д. 1",
			wantError:   false,
		},
		{
			name:        "postal address with label",
			attributes:  "Почтовый адрес: г. Санкт-Петербург, пр. Невский, д. 10",
			wantAddress: "г. Санкт-Петербург, пр. Невский, д. 10",
			wantError:   false,
		},
		{
			name:        "address in XML tag",
			attributes:  "<Адрес>г. Москва, ул. Пушкина, д. 5, кв. 10</Адрес>",
			wantAddress: ">г. Москва, ул. Пушкина, д. 5, кв. 10",
			wantError:   false,
		},
		{
			name:        "empty attributes",
			attributes:  "",
			wantAddress: "",
			wantError:   true,
		},
		{
			name:        "no address found",
			attributes:  "Some text without address",
			wantAddress: "",
			wantError:   true,
		},
		{
			name:        "address too short",
			attributes:  "Адрес: М",
			wantAddress: "",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractAddressFromAttributes(tt.attributes)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractAddressFromAttributes() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantAddress {
				t.Errorf("ExtractAddressFromAttributes() = %v, want %v", got, tt.wantAddress)
			}
		})
	}
}

func TestExtractContactPhoneFromAttributes(t *testing.T) {
	tests := []struct {
		name      string
		attributes string
		wantPhone string
		wantError bool
	}{
		{
			name:      "phone with label",
			attributes: "Телефон: +74951234567",
			wantPhone: "+74951234567",
			wantError: false,
		},
		{
			name:      "phone with label lowercase",
			attributes: "телефон: 8-800-555-35-35",
			wantPhone: "8-800-555-35-35",
			wantError: false,
		},
		{
			name:      "mobile phone",
			attributes: "Мобильный: +7 999 123 45 67",
			wantPhone: "+7 999 123 45 67",
			wantError: false,
		},
		{
			name:      "empty attributes",
			attributes: "",
			wantPhone:  "",
			wantError: true,
		},
		{
			name:      "no phone found",
			attributes: "Some text without phone",
			wantPhone:  "",
			wantError: true,
		},
		{
			name:      "phone too short",
			attributes: "Телефон: 123",
			wantPhone:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractContactPhoneFromAttributes(tt.attributes)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractContactPhoneFromAttributes() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantPhone {
				t.Errorf("ExtractContactPhoneFromAttributes() = %v, want %v", got, tt.wantPhone)
			}
		})
	}
}

func TestExtractContactEmailFromAttributes(t *testing.T) {
	tests := []struct {
		name      string
		attributes string
		wantEmail string
		wantError bool
	}{
		{
			name:      "email with label",
			attributes: "Email: test@example.com",
			wantEmail: "test@example.com",
			wantError: false,
		},
		{
			name:      "email with label lowercase",
			attributes: "email: contact@company.ru",
			wantEmail: "contact@company.ru",
			wantError: false,
		},
		{
			name:      "email as standalone",
			attributes: "Contact us at info@example.com for details",
			wantEmail: "info@example.com",
			wantError: false,
		},
		{
			name:      "empty attributes",
			attributes: "",
			wantEmail:  "",
			wantError: true,
		},
		{
			name:      "no email found",
			attributes: "Some text without email",
			wantEmail:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractContactEmailFromAttributes(tt.attributes)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractContactEmailFromAttributes() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantEmail {
				t.Errorf("ExtractContactEmailFromAttributes() = %v, want %v", got, tt.wantEmail)
			}
		})
	}
}

func TestExtractContactPersonFromAttributes(t *testing.T) {
	tests := []struct {
		name      string
		attributes string
		wantPerson string
		wantError  bool
	}{
		{
			name:       "contact person with label",
			attributes: "Контактное лицо: Иванов Иван Иванович",
			wantPerson: "Иванов Иван Иванович",
			wantError:  false,
		},
		{
			name:       "director with label",
			attributes: "Директор: Петров Петр Петрович",
			wantPerson: "Петров Петр Петрович",
			wantError:  false,
		},
		{
			name:       "empty attributes",
			attributes: "",
			wantPerson: "",
			wantError:  true,
		},
		{
			name:       "no person found",
			attributes: "Some text without person",
			wantPerson: "",
			wantError:  true,
		},
		{
			name:       "person too short",
			attributes: "Контактное лицо: Ив",
			wantPerson: "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractContactPersonFromAttributes(tt.attributes)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractContactPersonFromAttributes() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.wantPerson {
				t.Errorf("ExtractContactPersonFromAttributes() = %v, want %v", got, tt.wantPerson)
			}
		})
	}
}

func TestValidateCodeFormat(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		format string
		want   bool
	}{
		{
			name:   "numeric valid",
			code:   "123456",
			format: "numeric",
			want:   true,
		},
		{
			name:   "numeric invalid with letters",
			code:   "12345a",
			format: "numeric",
			want:   false,
		},
		{
			name:   "alphanumeric valid",
			code:   "ABC123",
			format: "alphanumeric",
			want:   true,
		},
		{
			name:   "alphanumeric invalid with special chars",
			code:   "ABC-123",
			format: "alphanumeric",
			want:   false,
		},
		{
			name:   "any format valid",
			code:   "ABC-123_xyz",
			format: "any",
			want:   true,
		},
		{
			name:   "any format empty",
			code:   "",
			format: "any",
			want:   false,
		},
		{
			name:   "empty code",
			code:   "",
			format: "numeric",
			want:   false,
		},
		{
			name:   "unknown format",
			code:   "123",
			format: "unknown",
			want:   true, // Should return true for non-empty code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCodeFormat(tt.code, tt.format)
			if got != tt.want {
				t.Errorf("ValidateCodeFormat(%q, %q) = %v, want %v", tt.code, tt.format, got, tt.want)
			}
		})
	}
}

