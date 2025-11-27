-- Миграция: Добавление расширенных полей для клиентов и таблицы документов
-- Дата создания: 2025-01-XX
-- Описание: Добавляет бизнес-информацию, расширенные контакты, юридические данные,
--           договорные данные и таблицу для документов клиентов

-- Добавление новых колонок в таблицу clients
-- Проверяем существование колонок перед добавлением

-- Бизнес-информация
ALTER TABLE clients ADD COLUMN IF NOT EXISTS industry TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS company_size TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS legal_form TEXT;

-- Расширенные контакты
ALTER TABLE clients ADD COLUMN IF NOT EXISTS contact_person TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS contact_position TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS alternate_phone TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS website TEXT;

-- Юридические данные
ALTER TABLE clients ADD COLUMN IF NOT EXISTS ogrn TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS kpp TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS legal_address TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS postal_address TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS bank_name TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS bank_account TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS correspondent_account TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS bik TEXT;

-- Договорные данные
ALTER TABLE clients ADD COLUMN IF NOT EXISTS contract_number TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS contract_date TIMESTAMP;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS contract_terms TEXT;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS contract_expires_at TIMESTAMP;

-- Создание таблицы документов клиента (если не существует)
CREATE TABLE IF NOT EXISTS client_documents (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  client_id INTEGER NOT NULL,
  file_name TEXT NOT NULL,
  file_path TEXT NOT NULL,
  file_type TEXT NOT NULL,
  file_size INTEGER NOT NULL,
  category TEXT DEFAULT 'technical',
  description TEXT,
  uploaded_by TEXT,
  uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(client_id) REFERENCES clients(id) ON DELETE CASCADE
);

-- Создание индексов для таблицы документов
CREATE INDEX IF NOT EXISTS idx_client_documents_client_id ON client_documents(client_id);
CREATE INDEX IF NOT EXISTS idx_client_documents_category ON client_documents(category);
CREATE INDEX IF NOT EXISTS idx_client_documents_uploaded_at ON client_documents(uploaded_at DESC);

