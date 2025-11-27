package server

import (
	"net/http"
)

// Обертки для методов Server, которые делегируют вызовы в uploadLegacyHandler
// Эти методы используются в server.go для регистрации маршрутов

// handleHandshake обрабатывает рукопожатие
func (s *Server) handleHandshake(w http.ResponseWriter, r *http.Request) {
	if s.uploadLegacyHandler != nil {
		s.uploadLegacyHandler.HandleHandshake(w, r)
	} else {
		http.Error(w, "Upload legacy handler not initialized", http.StatusInternalServerError)
	}
}

// handleMetadata обрабатывает метаинформацию
func (s *Server) handleMetadata(w http.ResponseWriter, r *http.Request) {
	if s.uploadLegacyHandler != nil {
		s.uploadLegacyHandler.HandleMetadata(w, r)
	} else {
		http.Error(w, "Upload legacy handler not initialized", http.StatusInternalServerError)
	}
}

// handleConstant обрабатывает константу
func (s *Server) handleConstant(w http.ResponseWriter, r *http.Request) {
	if s.uploadLegacyHandler != nil {
		s.uploadLegacyHandler.HandleConstant(w, r)
	} else {
		http.Error(w, "Upload legacy handler not initialized", http.StatusInternalServerError)
	}
}

// handleCatalogMeta обрабатывает метаданные справочника
func (s *Server) handleCatalogMeta(w http.ResponseWriter, r *http.Request) {
	if s.uploadLegacyHandler != nil {
		s.uploadLegacyHandler.HandleCatalogMeta(w, r)
	} else {
		http.Error(w, "Upload legacy handler not initialized", http.StatusInternalServerError)
	}
}

// handleCatalogItem обрабатывает элемент справочника
func (s *Server) handleCatalogItem(w http.ResponseWriter, r *http.Request) {
	if s.uploadLegacyHandler != nil {
		s.uploadLegacyHandler.HandleCatalogItem(w, r)
	} else {
		http.Error(w, "Upload legacy handler not initialized", http.StatusInternalServerError)
	}
}

// handleCatalogItems обрабатывает пакетную загрузку элементов справочника
func (s *Server) handleCatalogItems(w http.ResponseWriter, r *http.Request) {
	if s.uploadLegacyHandler != nil {
		s.uploadLegacyHandler.HandleCatalogItems(w, r)
	} else {
		http.Error(w, "Upload legacy handler not initialized", http.StatusInternalServerError)
	}
}

// handleNomenclatureBatch обрабатывает пакетную загрузку номенклатуры с характеристиками
func (s *Server) handleNomenclatureBatch(w http.ResponseWriter, r *http.Request) {
	if s.uploadLegacyHandler != nil {
		s.uploadLegacyHandler.HandleNomenclatureBatch(w, r)
	} else {
		http.Error(w, "Upload legacy handler not initialized", http.StatusInternalServerError)
	}
}

// handleComplete обрабатывает завершение выгрузки
func (s *Server) handleComplete(w http.ResponseWriter, r *http.Request) {
	if s.uploadLegacyHandler != nil {
		s.uploadLegacyHandler.HandleComplete(w, r)
	} else {
		http.Error(w, "Upload legacy handler not initialized", http.StatusInternalServerError)
	}
}
