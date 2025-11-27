// Централизованный экспорт всех типов

// Мониторинг
export * from './monitoring'

// Нормализация
export * from './normalization'

// Отчеты
export * from './reports'

// Клиенты и проекты
export interface Client {
  id: number
  name: string
  legal_name?: string
  description?: string
  contact_email?: string
  contact_phone?: string
  tax_id?: string
  status: string
  project_count?: number
  benchmark_count?: number
  last_activity?: string
  country?: string
  created_by?: string
  created_at?: string
  updated_at?: string
  // Бизнес-информация
  industry?: string
  company_size?: string
  legal_form?: string
  // Расширенные контакты
  contact_person?: string
  contact_position?: string
  alternate_phone?: string
  website?: string
  // Юридические данные
  ogrn?: string
  kpp?: string
  legal_address?: string
  postal_address?: string
  bank_name?: string
  bank_account?: string
  correspondent_account?: string
  bik?: string
  // Договорные данные
  contract_number?: string
  contract_date?: string
  contract_terms?: string
  contract_expires_at?: string
}

export interface ClientProject {
  id: number
  client_id: number
  name: string
  project_type: string
  description?: string
  source_system?: string
  status: string
  target_quality_score?: number
  created_at?: string
  updated_at?: string
}

export interface ClientStatistics {
  total_projects: number
  total_benchmarks: number
  active_sessions: number
  avg_quality_score: number
}

export interface ClientDetail {
  client: Client
  projects: ClientProject[]
  statistics: ClientStatistics
  documents?: ClientDocument[]
}

export interface ClientDocument {
  id: number
  client_id: number
  file_name: string
  file_path: string
  file_type: string
  file_size: number
  category: string
  description?: string
  uploaded_by?: string
  uploaded_at: string
}

// Базы данных
export interface Database {
  id: number
  name: string
  file_path: string
  type?: string
  is_active?: boolean
  client_id?: number
  project_id?: number
  created_at?: string
  updated_at?: string
}

export interface DatabaseInfo {
  name: string
  path: string
  status?: string
  lastUpdate?: string
  type?: string
}

// Проекты
export interface ProjectDatabase {
  id: number
  client_project_id: number
  name: string
  file_path: string
  is_active?: boolean
  last_session?: {
    id: number
    status: string
    created_at: string
    finished_at?: string | null
  }
}

