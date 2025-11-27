package server

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DatabaseConnectionCache кэш для подключений к базам данных
// Оптимизирует открытие БД в циклах, предотвращая множественные открытия/закрытия
type DatabaseConnectionCache struct {
	mu          sync.RWMutex
	connections map[string]*cachedDBConnection
	maxAge      time.Duration
}

// cachedDBConnection кэшированное подключение к БД
type cachedDBConnection struct {
	db       *sql.DB
	lastUsed time.Time
	refCount int
	mu       sync.RWMutex
}

// NewDatabaseConnectionCache создает новый кэш подключений к БД
func NewDatabaseConnectionCache() *DatabaseConnectionCache {
	return &DatabaseConnectionCache{
		connections: make(map[string]*cachedDBConnection),
		maxAge:      10 * time.Minute, // Максимальный возраст подключения
	}
}

// GetConnection получает подключение к БД из кэша или создает новое
// Возвращает *sql.DB для совместимости с существующим кодом
func (c *DatabaseConnectionCache) GetConnection(dbPath string) (*sql.DB, error) {
	c.mu.RLock()
	if conn, exists := c.connections[dbPath]; exists {
		conn.mu.Lock()
		// Проверяем, не устарело ли подключение
		if time.Since(conn.lastUsed) < c.maxAge {
			conn.lastUsed = time.Now()
			conn.refCount++
			conn.mu.Unlock()
			c.mu.RUnlock()
			return conn.db, nil
		}
		// Подключение устарело, закрываем его
		if closeErr := conn.db.Close(); closeErr != nil {
			// Логируем ошибку, но продолжаем
		}
		conn.mu.Unlock()
		c.mu.RUnlock()
		
		// Удаляем устаревшее подключение
		c.mu.Lock()
		delete(c.connections, dbPath)
		c.mu.Unlock()
	} else {
		c.mu.RUnlock()
	}

	// Создаем новое подключение
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbPath, err)
	}

	// Настройка connection pooling
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(10 * time.Minute)

	// Сохраняем в кэш
	c.mu.Lock()
	c.connections[dbPath] = &cachedDBConnection{
		db:       db,
		lastUsed: time.Now(),
		refCount: 1,
	}
	c.mu.Unlock()

	return db, nil
}

// ReleaseConnection уменьшает счетчик ссылок на подключение
// ВАЖНО: Не закрывает подключение, только уменьшает счетчик
func (c *DatabaseConnectionCache) ReleaseConnection(dbPath string) {
	c.mu.RLock()
	conn, exists := c.connections[dbPath]
	c.mu.RUnlock()

	if !exists {
		return
	}

	conn.mu.Lock()
	conn.refCount--
	if conn.refCount < 0 {
		conn.refCount = 0
	}
	conn.lastUsed = time.Now()
	conn.mu.Unlock()
}

// CloseConnection закрывает и удаляет подключение из кэша
func (c *DatabaseConnectionCache) CloseConnection(dbPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, exists := c.connections[dbPath]
	if !exists {
		return nil
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if err := conn.db.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	delete(c.connections, dbPath)
	return nil
}

// CleanupStaleConnections очищает устаревшие подключения из кэша
func (c *DatabaseConnectionCache) CleanupStaleConnections() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for path, conn := range c.connections {
		conn.mu.RLock()
		age := now.Sub(conn.lastUsed)
		refCount := conn.refCount
		conn.mu.RUnlock()

		// Удаляем подключения, которые не использовались более maxAge и не имеют активных ссылок
		if age > c.maxAge && refCount <= 0 {
			conn.mu.Lock()
			if closeErr := conn.db.Close(); closeErr != nil {
				// Логируем ошибку, но продолжаем
			}
			conn.mu.Unlock()
			delete(c.connections, path)
		}
	}
}

// CloseAll закрывает все подключения в кэше
func (c *DatabaseConnectionCache) CloseAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for path, conn := range c.connections {
		conn.mu.Lock()
		if err := conn.db.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close connection %s: %w", path, err)
		}
		conn.mu.Unlock()
	}

	c.connections = make(map[string]*cachedDBConnection)
	return lastErr
}


