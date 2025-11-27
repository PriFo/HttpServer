package cache

import (
	"sync"
	"time"

	"httpserver/database"
)

// DatabaseInfoCache кэш для информации о базе данных, проекте и клиенте
type DatabaseInfoCache struct {
	mu              sync.RWMutex
	dbInfoCache     map[int]*database.ProjectDatabase
	projectCache    map[int]*database.ClientProject
	clientCache     map[int]*database.Client
	dbInfoTTL       time.Duration
	projectTTL      time.Duration
	clientTTL       time.Duration
	dbInfoExpiry    map[int]time.Time
	projectExpiry   map[int]time.Time
	clientExpiry    map[int]time.Time
}

// NewDatabaseInfoCache создает новый кэш для информации о БД
func NewDatabaseInfoCache() *DatabaseInfoCache {
	return &DatabaseInfoCache{
		dbInfoCache:  make(map[int]*database.ProjectDatabase),
		projectCache: make(map[int]*database.ClientProject),
		clientCache:  make(map[int]*database.Client),
		dbInfoTTL:    5 * time.Minute,  // TTL для информации о БД
		projectTTL:   10 * time.Minute, // TTL для информации о проекте
		clientTTL:    15 * time.Minute, // TTL для информации о клиенте
		dbInfoExpiry: make(map[int]time.Time),
		projectExpiry: make(map[int]time.Time),
		clientExpiry:  make(map[int]time.Time),
	}
}

// GetProjectDatabase получает информацию о БД из кэша или БД
func (c *DatabaseInfoCache) GetProjectDatabase(serviceDB *database.ServiceDB, databaseID int) (*database.ProjectDatabase, error) {
	c.mu.RLock()
	if dbInfo, exists := c.dbInfoCache[databaseID]; exists {
		if expiry, ok := c.dbInfoExpiry[databaseID]; ok && time.Now().Before(expiry) {
			c.mu.RUnlock()
			return dbInfo, nil
		}
	}
	c.mu.RUnlock()

	// Кэш промах - получаем из БД
	dbInfo, err := serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	c.mu.Lock()
	c.dbInfoCache[databaseID] = dbInfo
	c.dbInfoExpiry[databaseID] = time.Now().Add(c.dbInfoTTL)
	c.mu.Unlock()

	return dbInfo, nil
}

// GetClientProject получает информацию о проекте из кэша или БД
func (c *DatabaseInfoCache) GetClientProject(serviceDB *database.ServiceDB, projectID int) (*database.ClientProject, error) {
	c.mu.RLock()
	if project, exists := c.projectCache[projectID]; exists {
		if expiry, ok := c.projectExpiry[projectID]; ok && time.Now().Before(expiry) {
			c.mu.RUnlock()
			return project, nil
		}
	}
	c.mu.RUnlock()

	// Кэш промах - получаем из БД
	project, err := serviceDB.GetClientProject(projectID)
	if err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	c.mu.Lock()
	c.projectCache[projectID] = project
	c.projectExpiry[projectID] = time.Now().Add(c.projectTTL)
	c.mu.Unlock()

	return project, nil
}

// GetClient получает информацию о клиенте из кэша или БД
func (c *DatabaseInfoCache) GetClient(serviceDB *database.ServiceDB, clientID int) (*database.Client, error) {
	c.mu.RLock()
	if client, exists := c.clientCache[clientID]; exists {
		if expiry, ok := c.clientExpiry[clientID]; ok && time.Now().Before(expiry) {
			c.mu.RUnlock()
			return client, nil
		}
	}
	c.mu.RUnlock()

	// Кэш промах - получаем из БД
	client, err := serviceDB.GetClient(clientID)
	if err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	c.mu.Lock()
	c.clientCache[clientID] = client
	c.clientExpiry[clientID] = time.Now().Add(c.clientTTL)
	c.mu.Unlock()

	return client, nil
}

// InvalidateProjectDatabase инвалидирует кэш для БД
func (c *DatabaseInfoCache) InvalidateProjectDatabase(databaseID int) {
	c.mu.Lock()
	delete(c.dbInfoCache, databaseID)
	delete(c.dbInfoExpiry, databaseID)
	c.mu.Unlock()
}

// InvalidateClientProject инвалидирует кэш для проекта
func (c *DatabaseInfoCache) InvalidateClientProject(projectID int) {
	c.mu.Lock()
	delete(c.projectCache, projectID)
	delete(c.projectExpiry, projectID)
	c.mu.Unlock()
}

// InvalidateClient инвалидирует кэш для клиента
func (c *DatabaseInfoCache) InvalidateClient(clientID int) {
	c.mu.Lock()
	delete(c.clientCache, clientID)
	delete(c.clientExpiry, clientID)
	c.mu.Unlock()
}

// Clear очищает весь кэш
func (c *DatabaseInfoCache) Clear() {
	c.mu.Lock()
	c.dbInfoCache = make(map[int]*database.ProjectDatabase)
	c.projectCache = make(map[int]*database.ClientProject)
	c.clientCache = make(map[int]*database.Client)
	c.dbInfoExpiry = make(map[int]time.Time)
	c.projectExpiry = make(map[int]time.Time)
	c.clientExpiry = make(map[int]time.Time)
	c.mu.Unlock()
}

