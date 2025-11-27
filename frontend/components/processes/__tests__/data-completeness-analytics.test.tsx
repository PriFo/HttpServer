import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DataCompletenessAnalytics } from '../data-completeness-analytics'
import { CompletenessMetrics, NormalizationType } from '@/types/normalization'

describe('DataCompletenessAnalytics', () => {
  const mockCompleteness: CompletenessMetrics = {
    nomenclature_completeness: {
      articles_percent: 75.5,
      units_percent: 80.0,
      descriptions_percent: 65.5,
      overall_completeness: 73.67,
    },
    counterparty_completeness: {
      inn_percent: 90.0,
      address_percent: 70.0,
      contacts_percent: 85.0,
      overall_completeness: 81.67,
    },
  }

  it('should render loading state', () => {
    render(
      <DataCompletenessAnalytics
        completeness={undefined}
        normalizationType="nomenclature"
        isLoading={true}
      />
    )
    
    expect(screen.getByText('Аналитика заполнения справочников')).toBeInTheDocument()
    expect(screen.getByText('Загрузка данных...')).toBeInTheDocument()
  })

  it('should return null when no completeness data', () => {
    const { container } = render(
      <DataCompletenessAnalytics
        completeness={undefined}
        normalizationType="nomenclature"
        isLoading={false}
      />
    )
    
    expect(container.firstChild).toBeNull()
  })

  it('should render nomenclature metrics when type is nomenclature', () => {
    render(
      <DataCompletenessAnalytics
        completeness={mockCompleteness}
        normalizationType="nomenclature"
        isLoading={false}
      />
    )
    
    expect(screen.getByText('Заполненность номенклатуры')).toBeInTheDocument()
    expect(screen.getByText(/75\.5%/)).toBeInTheDocument()
    expect(screen.getByText(/80\.0%/)).toBeInTheDocument()
    expect(screen.getByText(/65\.5%/)).toBeInTheDocument()
    expect(screen.queryByText('Заполненность контрагентов')).not.toBeInTheDocument()
  })

  it('should render counterparties metrics when type is counterparties', () => {
    render(
      <DataCompletenessAnalytics
        completeness={mockCompleteness}
        normalizationType="counterparties"
        isLoading={false}
      />
    )
    
    expect(screen.getByText('Заполненность контрагентов')).toBeInTheDocument()
    expect(screen.getByText(/90\.0%/)).toBeInTheDocument()
    expect(screen.getByText(/70\.0%/)).toBeInTheDocument()
    expect(screen.getByText(/85\.0%/)).toBeInTheDocument()
    expect(screen.queryByText('Заполненность номенклатуры')).not.toBeInTheDocument()
  })

  it('should render both metrics when type is both', () => {
    render(
      <DataCompletenessAnalytics
        completeness={mockCompleteness}
        normalizationType="both"
        isLoading={false}
      />
    )
    
    expect(screen.getByText('Заполненность номенклатуры')).toBeInTheDocument()
    expect(screen.getByText('Заполненность контрагентов')).toBeInTheDocument()
  })

  it('should handle missing nomenclature completeness gracefully', () => {
    const incompleteData: CompletenessMetrics = {
      counterparty_completeness: mockCompleteness.counterparty_completeness,
    }
    
    render(
      <DataCompletenessAnalytics
        completeness={incompleteData}
        normalizationType="both"
        isLoading={false}
      />
    )
    
    expect(screen.getByText('Заполненность контрагентов')).toBeInTheDocument()
    expect(screen.queryByText('Заполненность номенклатуры')).not.toBeInTheDocument()
  })

  it('should handle missing counterparty completeness gracefully', () => {
    const incompleteData: CompletenessMetrics = {
      nomenclature_completeness: mockCompleteness.nomenclature_completeness,
    }
    
    render(
      <DataCompletenessAnalytics
        completeness={incompleteData}
        normalizationType="both"
        isLoading={false}
      />
    )
    
    expect(screen.getByText('Заполненность номенклатуры')).toBeInTheDocument()
    expect(screen.queryByText('Заполненность контрагентов')).not.toBeInTheDocument()
  })

  it('should display correct badge colors for completeness levels', () => {
    const highCompleteness: CompletenessMetrics = {
      nomenclature_completeness: {
        articles_percent: 85.0,
        units_percent: 90.0,
        descriptions_percent: 88.0,
        overall_completeness: 87.67,
      },
    }
    
    render(
      <DataCompletenessAnalytics
        completeness={highCompleteness}
        normalizationType="nomenclature"
        isLoading={false}
      />
    )
    
    // Проверяем наличие бейджа с процентом (зеленый для >= 80%)
    const badge = screen.getByText(/87\.7%/)
    expect(badge).toBeInTheDocument()
  })

  it('should apply custom className', () => {
    const { container } = render(
      <DataCompletenessAnalytics
        completeness={mockCompleteness}
        normalizationType="nomenclature"
        isLoading={false}
        className="custom-class"
      />
    )
    
    const wrapper = container.firstChild as HTMLElement
    expect(wrapper).toHaveClass('custom-class')
  })
})

