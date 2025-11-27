package server

import (
	"net/http"
	"sync"

	"httpserver/database"
)

// ServerHandler предоставляет доступ к методам Server для обработчиков
type ServerHandler struct {
	// Методы для работы с базами данных
	GetDB              func() *database.DB
	GetNormalizedDB    func() *database.DB
	GetServiceDB       func() *database.ServiceDB
	GetCurrentDBPath   func() string
	GetDBMutex         func() *sync.RWMutex
	
	// Методы для JSON ответов
	WriteJSONResponse  func(w http.ResponseWriter, data interface{}, statusCode int)
	WriteJSONError     func(w http.ResponseWriter, message string, statusCode int)
	
	// Методы для валидации
	ValidateIntParam   func(r *http.Request, paramName string, defaultValue, min, max int) (int, error)
	ValidateIntPathParam func(paramStr, paramName string) (int, error)
	HandleValidationError func(w http.ResponseWriter, err error) bool
	
	// Методы для логирования
	LogErrorf          func(format string, args ...interface{})
}

// NewServerHandler создает новый ServerHandler
func NewServerHandler(
	getDB func() *database.DB,
	getNormalizedDB func() *database.DB,
	getServiceDB func() *database.ServiceDB,
	getCurrentDBPath func() string,
	getDBMutex func() *sync.RWMutex,
	writeJSONResponse func(w http.ResponseWriter, data interface{}, statusCode int),
	writeJSONError func(w http.ResponseWriter, message string, statusCode int),
	validateIntParam func(r *http.Request, paramName string, defaultValue, min, max int) (int, error),
	validateIntPathParam func(paramStr, paramName string) (int, error),
	handleValidationError func(w http.ResponseWriter, err error) bool,
	logErrorf func(format string, args ...interface{}),
) *ServerHandler {
	return &ServerHandler{
		GetDB:                getDB,
		GetNormalizedDB:      getNormalizedDB,
		GetServiceDB:         getServiceDB,
		GetCurrentDBPath:     getCurrentDBPath,
		GetDBMutex:           getDBMutex,
		WriteJSONResponse:    writeJSONResponse,
		WriteJSONError:       writeJSONError,
		ValidateIntParam:     validateIntParam,
		ValidateIntPathParam: validateIntPathParam,
		HandleValidationError: handleValidationError,
		LogErrorf:            logErrorf,
	}
}

