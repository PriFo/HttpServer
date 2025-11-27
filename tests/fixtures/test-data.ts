/**
 * Тестовые fixtures и данные
 * 
 * Предоставляет готовые тестовые данные для использования в тестах
 */

import * as fs from 'fs'
import * as path from 'path'

export interface TestDatabase {
  path: string
  name: string
  description: string
}

export interface TestClient {
  name: string
  legal_name: string
  description: string
  contact_email: string
  contact_phone: string
  tax_id?: string
}

export interface TestProject {
  name: string
  project_type: string
  description: string
  source_system: string
  target_quality_score: number
}

/**
 * Стандартные тестовые данные для клиента
 */
export function getTestClient(overrides?: Partial<TestClient>): TestClient {
  return {
    name: `Test Client ${Date.now()}`,
    legal_name: `Test Client Legal ${Date.now()}`,
    description: 'Тестовый клиент для E2E тестов',
    contact_email: 'test@example.com',
    contact_phone: '+79991234567',
    tax_id: '1234567890',
    ...overrides,
  }
}

/**
 * Стандартные тестовые данные для проекта
 */
export function getTestProject(overrides?: Partial<TestProject>): TestProject {
  return {
    name: `Test Project ${Date.now()}`,
    project_type: 'normalization',
    description: 'Тестовый проект для E2E тестов',
    source_system: '1C',
    target_quality_score: 0.9,
    ...overrides,
  }
}

/**
 * Список возможных путей к тестовым базам данных
 */
export const TEST_DATABASE_PATHS = [
  '1c_data.db',
  'data/1c_data.db',
  'test-data.db',
  'data/test-data.db',
  'tests/data/test-data.db',
  '../1c_data.db',
  '../data/1c_data.db',
]

/**
 * Находит тестовую базу данных
 */
export function findTestDatabase(): string | null {
  for (const dbPath of TEST_DATABASE_PATHS) {
    if (fs.existsSync(dbPath)) {
      return dbPath
    }
  }
  return null
}

/**
 * Создает временную тестовую базу данных (пустую)
 */
export function createTempTestDatabase(): string {
  const tempDir = path.join(__dirname, '../../temp')
  if (!fs.existsSync(tempDir)) {
    fs.mkdirSync(tempDir, { recursive: true })
  }

  const dbPath = path.join(tempDir, `test-${Date.now()}.db`)
  
  // Создаем пустую SQLite базу данных
  // В реальности можно использовать sqlite3 для создания структуры
  fs.writeFileSync(dbPath, Buffer.alloc(0))
  
  return dbPath
}

/**
 * Удаляет временную базу данных
 */
export function cleanupTempDatabase(dbPath: string): void {
  try {
    if (fs.existsSync(dbPath)) {
      fs.unlinkSync(dbPath)
    }
  } catch (error) {
    console.warn(`Failed to cleanup temp database ${dbPath}:`, error)
  }
}

/**
 * Генерирует уникальное имя для тестовых данных
 */
export function generateTestName(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
}

/**
 * Ожидает, пока условие не станет истинным
 */
export async function waitForCondition(
  condition: () => Promise<boolean> | boolean,
  timeout: number = 10000,
  interval: number = 500
): Promise<boolean> {
  const startTime = Date.now()
  
  while (Date.now() - startTime < timeout) {
    const result = await condition()
    if (result) {
      return true
    }
    await new Promise(resolve => setTimeout(resolve, interval))
  }
  
  return false
}

/**
 * Ожидает, пока элемент не появится на странице
 */
export async function waitForElement(
  page: any,
  selector: string,
  timeout: number = 10000
): Promise<boolean> {
  try {
    await page.waitForSelector(selector, { timeout })
    return true
  } catch {
    return false
  }
}

