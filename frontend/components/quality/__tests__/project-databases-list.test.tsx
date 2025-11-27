import type { ComponentProps } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ProjectDatabasesList } from '../project-databases-list'
import type { DatabaseStat } from '@/app/quality/page'

const mockDatabases: DatabaseStat[] = [
  {
    database_id: 1,
    database_name: 'Test Database 1',
    database_path: '/path/to/db1.db',
    stats: {
      total_items: 100,
      average_quality: 0.85,
      benchmark_count: 10,
      benchmark_percentage: 0.1,
      by_level: {
        basic: { count: 50, avg_quality: 0.5, percentage: 0.5 },
        ai_enhanced: { count: 40, avg_quality: 0.9, percentage: 0.4 },
        benchmark: { count: 10, avg_quality: 0.95, percentage: 0.1 },
      },
    },
  },
  {
    database_id: 2,
    database_name: 'Test Database 2',
    database_path: '/path/to/db2.db',
    stats: {
      total_items: 200,
      average_quality: 0.75,
      benchmark_count: 20,
      benchmark_percentage: 0.1,
      by_level: {
        basic: { count: 100, avg_quality: 0.5, percentage: 0.5 },
        ai_enhanced: { count: 80, avg_quality: 0.8, percentage: 0.4 },
        benchmark: { count: 20, avg_quality: 0.95, percentage: 0.1 },
      },
    },
  },
  {
    database_id: 3,
    database_name: 'Another Database',
    database_path: '/path/to/db3.db',
    stats: {
      total_items: 50,
      average_quality: 0.95,
      benchmark_count: 30,
      benchmark_percentage: 0.6,
      by_level: {
        basic: { count: 10, avg_quality: 0.5, percentage: 0.2 },
        ai_enhanced: { count: 10, avg_quality: 0.9, percentage: 0.2 },
        benchmark: { count: 30, avg_quality: 0.95, percentage: 0.6 },
      },
    },
  },
]

const renderComponent = (overrideProps: Partial<ComponentProps<typeof ProjectDatabasesList>> = {}) =>
  render(
    <ProjectDatabasesList
      databases={mockDatabases}
      onDatabaseSelect={vi.fn()}
      {...overrideProps}
    />,
  )

describe('ProjectDatabasesList', () => {
  const mockOnDatabaseSelect = vi.fn()

  beforeEach(() => {
    mockOnDatabaseSelect.mockReset()
  })

  it('renders loading state skeleton', () => {
    renderComponent({ databases: [], loading: true, onDatabaseSelect: mockOnDatabaseSelect })

    expect(screen.getByText('Загрузка статистики...')).toBeInTheDocument()
    expect(screen.getByText('Обработка баз данных проекта...')).toBeInTheDocument()
  })

  it('renders empty state when нет активных баз данных', () => {
    renderComponent({ databases: [], loading: false, onDatabaseSelect: mockOnDatabaseSelect })

    expect(screen.getByText('В проекте нет активных баз данных')).toBeInTheDocument()
  })

  it('renders databases list and highlights selected database', () => {
    renderComponent({ selectedDatabase: '/path/to/db1.db', onDatabaseSelect: mockOnDatabaseSelect })

    expect(screen.getByText('Test Database 1')).toBeInTheDocument()
    expect(screen.getByText('Test Database 2')).toBeInTheDocument()
    expect(screen.getByText('Another Database')).toBeInTheDocument()
    expect(screen.getByText('Выбрана')).toBeInTheDocument()
  })

  it('calls onDatabaseSelect when card clicked', () => {
    renderComponent({ onDatabaseSelect: mockOnDatabaseSelect })

    fireEvent.click(screen.getByText('Another Database'))
    expect(mockOnDatabaseSelect).toHaveBeenCalledWith('/path/to/db3.db')
  })

  it('filters databases by search query', async () => {
    renderComponent({ onDatabaseSelect: mockOnDatabaseSelect })

    const searchInput = screen.getByPlaceholderText('Поиск по имени или пути...')
    fireEvent.change(searchInput, { target: { value: 'another' } })

    await waitFor(() => {
      expect(screen.getByText('Another Database')).toBeInTheDocument()
      expect(screen.queryByText('Test Database 1')).not.toBeInTheDocument()
      expect(screen.queryByText('Test Database 2')).not.toBeInTheDocument()
      expect(screen.getByText(/отфильтровано/)).toBeInTheDocument()
    })
  })

  it('shows empty state when filters return no results and allows clearing', async () => {
    renderComponent({ onDatabaseSelect: mockOnDatabaseSelect })

    const searchInput = screen.getByPlaceholderText('Поиск по имени или пути...')
    fireEvent.change(searchInput, { target: { value: 'missing-db' } })

    await waitFor(() => {
      expect(screen.getByText('Базы данных не найдены')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Очистить фильтры' }))

    await waitFor(() => {
      expect(screen.getByText('Test Database 1')).toBeInTheDocument()
      expect(screen.getByText('Test Database 2')).toBeInTheDocument()
      expect(screen.getByText('Another Database')).toBeInTheDocument()
    })
  })
})

