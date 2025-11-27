package server

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

// TestHandshakeRequest проверяет структуру запроса рукопожатия
func TestHandshakeRequest(t *testing.T) {
	req := HandshakeRequest{
		DatabaseID:      "test-db-id",
		Version1C:       "8.3.20",
		ConfigName:      "test-config",
		ConfigVersion:   "1.0",
		ComputerName:    "test-computer",
		UserName:        "test-user",
		Timestamp:       time.Now().Format(time.RFC3339),
		IterationNumber: 1,
		IterationLabel:  "test-iteration",
		ProgrammerName:  "test-programmer",
		UploadPurpose:   "test-purpose",
		ParentUploadID:  "parent-uuid",
	}
	
	if req.Version1C == "" {
		t.Error("HandshakeRequest.Version1C should not be empty")
	}
	
	if req.ConfigName == "" {
		t.Error("HandshakeRequest.ConfigName should not be empty")
	}
	
	if req.Timestamp == "" {
		t.Error("HandshakeRequest.Timestamp should not be empty")
	}
}

// TestHandshakeResponse проверяет структуру ответа на рукопожатие
func TestHandshakeResponse(t *testing.T) {
	resp := HandshakeResponse{
		Success:      true,
		UploadUUID:   "test-uuid",
		ClientName:   "test-client",
		ProjectName:  "test-project",
		DatabaseName: "test-database",
		Message:      "test message",
		Timestamp:    time.Now().Format(time.RFC3339),
	}
	
	if resp.UploadUUID == "" {
		t.Error("HandshakeResponse.UploadUUID should not be empty")
	}
	
	if resp.Message == "" {
		t.Error("HandshakeResponse.Message should not be empty")
	}
	
	if resp.Timestamp == "" {
		t.Error("HandshakeResponse.Timestamp should not be empty")
	}
}

// TestMetadataRequest проверяет структуру запроса метаинформации
func TestMetadataRequest(t *testing.T) {
	req := MetadataRequest{
		UploadUUID:    "test-uuid",
		DatabaseID:    "test-db-id",
		Version1C:     "8.3.20",
		ConfigName:    "test-config",
		ConfigVersion: "1.0",
		ComputerName:  "test-computer",
		UserName:      "test-user",
		Timestamp:     time.Now().Format(time.RFC3339),
	}
	
	if req.UploadUUID == "" {
		t.Error("MetadataRequest.UploadUUID should not be empty")
	}
	
	if req.Version1C == "" {
		t.Error("MetadataRequest.Version1C should not be empty")
	}
	
	if req.ConfigName == "" {
		t.Error("MetadataRequest.ConfigName should not be empty")
	}
}

// TestMetadataResponse проверяет структуру ответа на метаинформацию
func TestMetadataResponse(t *testing.T) {
	resp := MetadataResponse{
		Success:   true,
		Message:   "test message",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	
	if resp.Message == "" {
		t.Error("MetadataResponse.Message should not be empty")
	}
	
	if resp.Timestamp == "" {
		t.Error("MetadataResponse.Timestamp should not be empty")
	}
}

// TestConstantValue_UnmarshalXML проверяет парсинг вложенного XML
func TestConstantValue_UnmarshalXML(t *testing.T) {
	xmlStr := `<value><nested attr="test">content</nested></value>`
	
	var cv ConstantValue
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	
	// Пропускаем до начала тега value
	for {
		token, err := decoder.Token()
		if err != nil {
			t.Fatalf("Failed to read token: %v", err)
		}
		
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "value" {
			err = cv.UnmarshalXML(decoder, start)
			if err != nil {
				t.Fatalf("UnmarshalXML failed: %v", err)
			}
			break
		}
	}
	
	if cv.Content == "" {
		t.Error("ConstantValue.Content should not be empty after unmarshaling")
	}
	
	if !strings.Contains(cv.Content, "nested") {
		t.Error("ConstantValue.Content should contain nested elements")
	}
}

// TestConstantValue_Empty проверяет обработку пустого значения
func TestConstantValue_Empty(t *testing.T) {
	xmlStr := `<value></value>`
	
	var cv ConstantValue
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	
	for {
		token, err := decoder.Token()
		if err != nil {
			t.Fatalf("Failed to read token: %v", err)
		}
		
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "value" {
			err = cv.UnmarshalXML(decoder, start)
			if err != nil {
				t.Fatalf("UnmarshalXML failed: %v", err)
			}
			break
		}
	}
	
	// Пустое значение допустимо
	_ = cv.Content
}

// TestHandshakeRequest_XML проверяет XML сериализацию запроса
func TestHandshakeRequest_XML(t *testing.T) {
	req := HandshakeRequest{
		Version1C:     "8.3.20",
		ConfigName:    "test-config",
		Timestamp:     time.Now().Format(time.RFC3339),
	}
	
	// Проверяем, что структура имеет правильные XML теги
	data, err := xml.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal HandshakeRequest: %v", err)
	}
	
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "handshake") {
		t.Error("XML should contain 'handshake' tag")
	}
	
	if !strings.Contains(xmlStr, "version_1c") {
		t.Error("XML should contain 'version_1c' field")
	}
}

// TestHandshakeResponse_XML проверяет XML сериализацию ответа
func TestHandshakeResponse_XML(t *testing.T) {
	resp := HandshakeResponse{
		Success:    true,
		UploadUUID: "test-uuid",
		Message:    "test message",
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	
	data, err := xml.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal HandshakeResponse: %v", err)
	}
	
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "handshake_response") {
		t.Error("XML should contain 'handshake_response' tag")
	}
	
	if !strings.Contains(xmlStr, "upload_uuid") {
		t.Error("XML should contain 'upload_uuid' field")
	}
}

