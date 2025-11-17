'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Database, Table, Columns, Save, Loader2, Settings2 } from 'lucide-react'
import { Label } from '@/components/ui/label'

interface NormalizationConfig {
  id: number
  database_path: string
  source_table: string
  reference_column: string
  code_column: string
  name_column: string
}

interface DatabaseInfo {
  path: string
  name: string
  type: string
}

interface TableInfo {
  name: string
  count: number
}

interface ColumnInfo {
  name: string
  type: string
  primary_key: boolean
  not_null: boolean
}

export function DataSourceSelector({ disabled }: { disabled?: boolean }) {
  const [config, setConfig] = useState<NormalizationConfig | null>(null)
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [tables, setTables] = useState<TableInfo[]>([])
  const [columns, setColumns] = useState<ColumnInfo[]>([])

  const [selectedDatabase, setSelectedDatabase] = useState('')
  const [selectedTable, setSelectedTable] = useState('')
  const [selectedRefColumn, setSelectedRefColumn] = useState('')
  const [selectedCodeColumn, setSelectedCodeColumn] = useState('')
  const [selectedNameColumn, setSelectedNameColumn] = useState('')

  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  // Загрузка текущей конфигурации
  useEffect(() => {
    loadConfig()
    loadDatabases()
  }, [])

  // Загрузка таблиц при выборе БД
  useEffect(() => {
    if (selectedDatabase) {
      loadTables(selectedDatabase)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedDatabase])

  // Загрузка колонок при выборе таблицы
  useEffect(() => {
    if (selectedTable) {
      loadColumns(selectedDatabase, selectedTable)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedTable])

  const loadConfig = async () => {
    try {
      const response = await fetch('/api/normalization/config')
      if (response.ok) {
        const data = await response.json()
        setConfig(data)
        setSelectedDatabase(data.database_path || '')
        setSelectedTable(data.source_table || '')
        setSelectedRefColumn(data.reference_column || '')
        setSelectedCodeColumn(data.code_column || '')
        setSelectedNameColumn(data.name_column || '')
      }
    } catch (error) {
      console.error('Error loading config:', error)
    }
  }

  const loadDatabases = async () => {
    try {
      const response = await fetch('/api/normalization/databases')
      if (response.ok) {
        const data = await response.json()
        setDatabases(data)
      }
    } catch (error) {
      console.error('Error loading databases:', error)
    }
  }

  const loadTables = async (database: string) => {
    setLoading(true)
    try {
      const url = database
        ? `/api/normalization/tables?database=${encodeURIComponent(database)}`
        : '/api/normalization/tables'
      const response = await fetch(url)
      if (response.ok) {
        const data = await response.json()
        setTables(data)
      }
    } catch (error) {
      console.error('Error loading tables:', error)
    } finally {
      setLoading(false)
    }
  }

  const loadColumns = async (database: string, table: string) => {
    setLoading(true)
    try {
      let url = `/api/normalization/columns?table=${encodeURIComponent(table)}`
      if (database) {
        url += `&database=${encodeURIComponent(database)}`
      }
      const response = await fetch(url)
      if (response.ok) {
        const data = await response.json()
        setColumns(data)
      }
    } catch (error) {
      console.error('Error loading columns:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSaveConfig = async () => {
    if (!selectedTable || !selectedRefColumn || !selectedCodeColumn || !selectedNameColumn) {
      alert('Заполните все поля')
      return
    }

    setSaving(true)
    try {
      const response = await fetch('/api/normalization/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          database_path: selectedDatabase,
          source_table: selectedTable,
          reference_column: selectedRefColumn,
          code_column: selectedCodeColumn,
          name_column: selectedNameColumn,
        }),
      })

      if (response.ok) {
        const data = await response.json()
        setConfig(data)
        alert('Конфигурация сохранена успешно!')
      } else {
        const error = await response.json()
        alert(`Ошибка сохранения: ${error.error}`)
      }
    } catch (error) {
      console.error('Error saving config:', error)
      alert('Ошибка сохранения конфигурации')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Settings2 className="h-5 w-5" />
          Настройка источника данных
        </CardTitle>
        <CardDescription>
          Выберите базу данных, таблицу и колонки для нормализации
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Выбор базы данных */}
        <div className="space-y-2">
          <Label className="flex items-center gap-2">
            <Database className="h-4 w-4" />
            База данных
          </Label>
          <Select
            value={selectedDatabase}
            onValueChange={setSelectedDatabase}
            disabled={disabled}
          >
            <SelectTrigger>
              <SelectValue placeholder="Выберите базу данных" />
            </SelectTrigger>
            <SelectContent>
              {databases.map((db) => (
                <SelectItem key={db.path} value={db.path}>
                  {db.name} {db.type === 'current' && '(текущая)'}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Выбор таблицы */}
        <div className="space-y-2">
          <Label className="flex items-center gap-2">
            <Table className="h-4 w-4" />
            Таблица
          </Label>
          <Select
            value={selectedTable}
            onValueChange={setSelectedTable}
            disabled={disabled || !selectedDatabase || loading}
          >
            <SelectTrigger>
              <SelectValue placeholder={loading ? "Загрузка..." : "Выберите таблицу"} />
            </SelectTrigger>
            <SelectContent>
              {tables.map((table) => (
                <SelectItem key={table.name} value={table.name}>
                  {table.name} ({table.count} записей)
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Выбор колонок */}
        {columns.length > 0 && (
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label>Reference (ID)</Label>
              <Select
                value={selectedRefColumn}
                onValueChange={setSelectedRefColumn}
                disabled={disabled}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Колонка ID" />
                </SelectTrigger>
                <SelectContent>
                  {columns.map((col) => (
                    <SelectItem key={col.name} value={col.name}>
                      {col.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Code (Код)</Label>
              <Select
                value={selectedCodeColumn}
                onValueChange={setSelectedCodeColumn}
                disabled={disabled}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Колонка кода" />
                </SelectTrigger>
                <SelectContent>
                  {columns.map((col) => (
                    <SelectItem key={col.name} value={col.name}>
                      {col.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Name (Название)</Label>
              <Select
                value={selectedNameColumn}
                onValueChange={setSelectedNameColumn}
                disabled={disabled}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Колонка названия" />
                </SelectTrigger>
                <SelectContent>
                  {columns.map((col) => (
                    <SelectItem key={col.name} value={col.name}>
                      {col.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        )}

        {/* Кнопка сохранения */}
        <Button
          onClick={handleSaveConfig}
          disabled={disabled || saving || !selectedTable || !selectedRefColumn || !selectedCodeColumn || !selectedNameColumn}
          className="w-full"
        >
          {saving ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Сохранение...
            </>
          ) : (
            <>
              <Save className="mr-2 h-4 w-4" />
              Сохранить конфигурацию
            </>
          )}
        </Button>
      </CardContent>
    </Card>
  )
}
