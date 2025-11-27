/**
 * Утилиты для работы с тестовыми базами данных
 * 
 * Предоставляет функции для создания, проверки и копирования тестовых SQLite баз данных
 * без использования нативных зависимостей
 */

import * as fs from 'fs'
import * as path from 'path'

/**
 * Создает тестовую SQLite базу данных
 * Использует существующую БД как шаблон, если доступна, иначе создает минимальную
 * 
 * @param dbPath - Путь к файлу базы данных
 * @param templatePath - Опциональный путь к шаблону БД для копирования
 * @returns Promise<void>
 */
export async function createTestDatabase(dbPath: string, templatePath?: string): Promise<void> {
  // Создаем директорию, если она не существует
  const dbDir = path.dirname(dbPath)
  if (!fs.existsSync(dbDir)) {
    fs.mkdirSync(dbDir, { recursive: true })
  }

  // Проверяем, существует ли уже файл
  if (fs.existsSync(dbPath)) {
    console.log(`База данных уже существует: ${dbPath}`)
    return
  }

  // Если указан шаблон и он существует, копируем его
  if (templatePath && fs.existsSync(templatePath)) {
    if (isValidSQLiteFile(templatePath)) {
      await copyDatabase(templatePath, dbPath)
      console.log(`Создана тестовая БД из шаблона: ${dbPath}`)
      return
    } else {
      console.warn(`Шаблон ${templatePath} не является валидным SQLite файлом, создаем минимальную БД`)
    }
  }

  // Ищем существующую тестовую БД в стандартных местах
  const possibleTemplates = [
    '1c_data.db',
    'data/1c_data.db',
    'test-data.db',
    'data/test-data.db',
    'tests/data/test-data.db',
  ]

  for (const template of possibleTemplates) {
    if (fs.existsSync(template) && isValidSQLiteFile(template)) {
      await copyDatabase(template, dbPath)
      console.log(`Создана тестовая БД из найденного шаблона: ${template} -> ${dbPath}`)
      return
    }
  }

  // Если шаблон не найден, создаем минимальную SQLite базу данных
  // SQLite файл начинается с заголовка "SQLite format 3"
  const sqliteHeader = Buffer.from('SQLite format 3\x00')
  const emptyDbBuffer = Buffer.alloc(1024) // Минимальный размер SQLite файла
  sqliteHeader.copy(emptyDbBuffer, 0)

  // Записываем файл
  fs.writeFileSync(dbPath, emptyDbBuffer)

  console.log(`Создана минимальная тестовая база данных: ${dbPath}`)
  console.warn('⚠️ Для полноценной БД рекомендуется использовать существующую тестовую БД или создавать через API')
}

/**
 * Проверяет целостность базы данных
 * Проверяет наличие файла и его формат (SQLite заголовок)
 * 
 * @param dbPath - Путь к файлу базы данных
 * @returns Promise<boolean>
 */
export async function checkDatabaseIntegrity(dbPath: string): Promise<boolean> {
  if (!fs.existsSync(dbPath)) {
    console.warn(`Файл базы данных не найден: ${dbPath}`)
    return false
  }

  try {
    // Читаем первые байты файла для проверки формата SQLite
    const fileBuffer = fs.readFileSync(dbPath, { start: 0, end: 15 })
    const sqliteHeader = Buffer.from('SQLite format 3\x00')

    // Проверяем заголовок SQLite
    const hasValidHeader = fileBuffer.slice(0, sqliteHeader.length).equals(sqliteHeader)

    if (!hasValidHeader) {
      console.warn(`Файл ${dbPath} не является валидным SQLite файлом`)
      return false
    }

    // Проверяем размер файла (SQLite файл должен быть минимум 512 байт)
    const stats = fs.statSync(dbPath)
    if (stats.size < 512) {
      console.warn(`Файл ${dbPath} слишком мал для SQLite базы данных`)
      return false
    }

    return true
  } catch (error) {
    console.error(`Ошибка при проверке целостности БД ${dbPath}:`, error)
    return false
  }
}

/**
 * Получает статистику базы данных
 * Возвращает информацию о файле (размер, дата изменения)
 * 
 * @param dbPath - Путь к файлу базы данных
 * @returns Promise<{ tableCount: number, recordCount: number, fileSize: number }>
 */
export async function getDatabaseStats(dbPath: string): Promise<{
  tableCount: number
  recordCount: number
  fileSize: number
  lastModified: Date
}> {
  if (!fs.existsSync(dbPath)) {
    throw new Error(`Файл базы данных не найден: ${dbPath}`)
  }

  const stats = fs.statSync(dbPath)

  // Без доступа к sqlite3 мы не можем получить точное количество таблиц и записей
  // Возвращаем информацию о файле
  return {
    tableCount: 0, // Неизвестно без sqlite3
    recordCount: 0, // Неизвестно без sqlite3
    fileSize: stats.size,
    lastModified: stats.mtime,
  }
}

/**
 * Копирует базу данных
 * 
 * @param sourcePath - Путь к исходному файлу
 * @param targetPath - Путь к целевому файлу
 * @returns Promise<void>
 */
export async function copyDatabase(sourcePath: string, targetPath: string): Promise<void> {
  if (!fs.existsSync(sourcePath)) {
    throw new Error(`Исходный файл не найден: ${sourcePath}`)
  }

  // Создаем директорию для целевого файла, если не существует
  const targetDir = path.dirname(targetPath)
  if (!fs.existsSync(targetDir)) {
    fs.mkdirSync(targetDir, { recursive: true })
  }

  // Копируем файл
  fs.copyFileSync(sourcePath, targetPath)
  console.log(`База данных скопирована: ${sourcePath} -> ${targetPath}`)
}

/**
 * Удаляет базу данных
 * 
 * @param dbPath - Путь к файлу базы данных
 * @returns Promise<void>
 */
export async function deleteDatabase(dbPath: string): Promise<void> {
  if (fs.existsSync(dbPath)) {
    fs.unlinkSync(dbPath)
    console.log(`База данных удалена: ${dbPath}`)
  }
}

/**
 * Проверяет, является ли файл валидной SQLite базой данных
 * 
 * @param filePath - Путь к файлу
 * @returns boolean
 */
export function isValidSQLiteFile(filePath: string): boolean {
  if (!fs.existsSync(filePath)) {
    return false
  }

  try {
    const fileBuffer = fs.readFileSync(filePath, { start: 0, end: 15 })
    const sqliteHeader = Buffer.from('SQLite format 3\x00')
    return fileBuffer.slice(0, sqliteHeader.length).equals(sqliteHeader)
  } catch {
    return false
  }
}
