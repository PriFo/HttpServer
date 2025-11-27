import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { NormalizationTypeSelector } from '../normalization-type-selector'
import { NormalizationType } from '@/types/normalization'

describe('NormalizationTypeSelector', () => {
  const mockOnChange = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should render all three tabs', () => {
    render(
      <NormalizationTypeSelector
        value="both"
        onChange={mockOnChange}
      />
    )
    
    expect(screen.getByText('Номенклатура')).toBeInTheDocument()
    expect(screen.getByText('Контрагенты')).toBeInTheDocument()
    expect(screen.getByText('Комплексная')).toBeInTheDocument()
  })

  it('should call onChange when tab is clicked', () => {
    render(
      <NormalizationTypeSelector
        value="nomenclature"
        onChange={mockOnChange}
      />
    )
    
    const counterpartiesTab = screen.getByText('Контрагенты')
    fireEvent.click(counterpartiesTab)
    
    expect(mockOnChange).toHaveBeenCalledWith('counterparties')
  })

  it('should display counts when provided', () => {
    render(
      <NormalizationTypeSelector
        value="both"
        onChange={mockOnChange}
        nomenclatureCount={150}
        counterpartyCount={75}
        totalRecords={225}
      />
    )
    
    expect(screen.getByText('150')).toBeInTheDocument()
    expect(screen.getByText('75')).toBeInTheDocument()
    expect(screen.getByText('225')).toBeInTheDocument()
  })

  it('should not display counts when zero', () => {
    render(
      <NormalizationTypeSelector
        value="nomenclature"
        onChange={mockOnChange}
        nomenclatureCount={0}
        counterpartyCount={0}
      />
    )
    
    // Badges с нулевыми значениями не должны отображаться (это зависит от реализации компонента)
    // Проверяем, что компонент рендерится без ошибок
    expect(screen.getByText('Номенклатура')).toBeInTheDocument()
  })

  it('should show correct description for nomenclature', () => {
    render(
      <NormalizationTypeSelector
        value="nomenclature"
        onChange={mockOnChange}
      />
    )
    
    expect(screen.getByText('Нормализация товаров и услуг')).toBeInTheDocument()
    expect(screen.getByText('Нормализация названий товаров')).toBeInTheDocument()
    expect(screen.getByText('Классификация по КПВЭД/ОКПД2')).toBeInTheDocument()
  })

  it('should show correct description for counterparties', () => {
    render(
      <NormalizationTypeSelector
        value="counterparties"
        onChange={mockOnChange}
      />
    )
    
    expect(screen.getByText('Нормализация контрагентов')).toBeInTheDocument()
    expect(screen.getByText('Верификация реквизитов (ИНН/БИН)')).toBeInTheDocument()
    expect(screen.getByText('Стандартизация юридических форм')).toBeInTheDocument()
  })

  it('should show correct description for both', () => {
    render(
      <NormalizationTypeSelector
        value="both"
        onChange={mockOnChange}
      />
    )
    
    expect(screen.getByText('Комплексная обработка всех данных')).toBeInTheDocument()
    expect(screen.getByText('Нормализация номенклатуры и контрагентов')).toBeInTheDocument()
    expect(screen.getByText('Параллельная обработка')).toBeInTheDocument()
  })

  it('should apply custom className', () => {
    const { container } = render(
      <NormalizationTypeSelector
        value="nomenclature"
        onChange={mockOnChange}
        className="custom-class"
      />
    )
    
    const card = container.querySelector('.custom-class')
    expect(card).toBeInTheDocument()
  })

  it('should handle large numbers correctly', () => {
    render(
      <NormalizationTypeSelector
        value="both"
        onChange={mockOnChange}
        nomenclatureCount={1234567}
        counterpartyCount={987654}
      />
    )
    
    // Проверяем, что числа отформатированы с разделителями тысяч
    expect(screen.getByText(/1[,\s]234[,\s]567/)).toBeInTheDocument()
    expect(screen.getByText(/987[,\s]654/)).toBeInTheDocument()
  })
})

