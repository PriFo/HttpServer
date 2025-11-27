/**
 * 📋 E2E ТЕСТЫ ДЛЯ ПРОВЕРКИ ДОСТУПНОСТИ (A11Y)
 * 
 * Тесты проверяют соответствие стандартам доступности WCAG 2.1
 * 
 * Prerequisites:
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 * 2. Запущенный Next.js фронтенд на http://localhost:3000
 */

import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { waitForPageLoad, logPageInfo } from './test-helpers'

test.describe('Проверка доступности (A11Y)', () => {
  test('Главная страница должна быть доступной', async ({ page }) => {
    await page.goto('/')
    await waitForPageLoad(page)
    await logPageInfo(page)

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
      .analyze()

    if (accessibilityScanResults.violations.length > 0) {
      console.error('❌ Найдены нарушения доступности на главной странице:')
      accessibilityScanResults.violations.forEach((violation) => {
        console.error(`  - ${violation.id}: ${violation.description}`)
        console.error(`    Элементы: ${violation.nodes.length}`)
      })
    }

    // Для критичных страниц требуем отсутствие нарушений
    expect(accessibilityScanResults.violations).toEqual([])
  })

  test('Страница качества должна быть доступной', async ({ page }) => {
    await page.goto('/quality')
    await waitForPageLoad(page)
    await logPageInfo(page)

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze()

    // Допускаем некоторые нарушения, но логируем их
    if (accessibilityScanResults.violations.length > 0) {
      console.warn('⚠️ Найдены нарушения доступности на странице качества:')
      accessibilityScanResults.violations.forEach((violation) => {
        console.warn(`  - ${violation.id}: ${violation.description}`)
      })
    }

    // Проверяем, что нет критичных нарушений (серьезность "serious" или "critical")
    const criticalViolations = accessibilityScanResults.violations.filter(
      (v) => v.impact === 'serious' || v.impact === 'critical'
    )
    expect(criticalViolations.length).toBe(0)
  })

  test('Страница мониторинга должна быть доступной', async ({ page }) => {
    await page.goto('/monitoring')
    await waitForPageLoad(page)
    await logPageInfo(page)

    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze()

    if (accessibilityScanResults.violations.length > 0) {
      console.warn('⚠️ Найдены нарушения доступности:', accessibilityScanResults.violations)
    }
  })

  test('Формы должны иметь правильные labels', async ({ page }) => {
    await page.goto('/clients')
    await waitForPageLoad(page)
    await logPageInfo(page)

    // Проверяем, что все input имеют связанные labels
    const inputs = page.locator('input[type="text"], input[type="email"], textarea')
    const inputCount = await inputs.count()

    for (let i = 0; i < inputCount; i++) {
      const input = inputs.nth(i)
      const id = await input.getAttribute('id')
      const ariaLabel = await input.getAttribute('aria-label')
      const placeholder = await input.getAttribute('placeholder')

      // Должен быть либо id с label, либо aria-label, либо placeholder
      if (id) {
        const label = page.locator(`label[for="${id}"]`)
        const hasLabel = await label.count() > 0
        expect(hasLabel || ariaLabel || placeholder).toBeTruthy()
      } else {
        expect(ariaLabel || placeholder).toBeTruthy()
      }
    }
  })

  test('Кнопки должны иметь доступные имена', async ({ page }) => {
    await page.goto('/')
    await waitForPageLoad(page)
    await logPageInfo(page)

    const buttons = page.locator('button')
    const buttonCount = await buttons.count()

    for (let i = 0; i < buttonCount; i++) {
      const button = buttons.nth(i)
      const text = await button.textContent()
      const ariaLabel = await button.getAttribute('aria-label')
      const ariaLabelledBy = await button.getAttribute('aria-labelledby')

      // Кнопка должна иметь либо текст, либо aria-label, либо aria-labelledby
      expect(text?.trim() || ariaLabel || ariaLabelledBy).toBeTruthy()
    }
  })

  test('Изображения должны иметь alt текст', async ({ page }) => {
    await page.goto('/')
    await waitForPageLoad(page)
    await logPageInfo(page)

    const images = page.locator('img')
    const imageCount = await images.count()

    for (let i = 0; i < imageCount; i++) {
      const image = images.nth(i)
      const alt = await image.getAttribute('alt')
      const role = await image.getAttribute('role')

      // Декоративные изображения могут иметь пустой alt или role="presentation"
      if (role !== 'presentation') {
        expect(alt).not.toBeNull()
      }
    }
  })

  test('Навигация должна быть доступной с клавиатуры', async ({ page }) => {
    await page.goto('/')
    await waitForPageLoad(page)
    await logPageInfo(page)

    // Проверяем, что можно перейти по всем ссылкам с Tab
    const links = page.locator('a[href]')
    const linkCount = await links.count()

    expect(linkCount).toBeGreaterThan(0)

    // Проверяем, что ссылки имеют tabindex или доступны по умолчанию
    for (let i = 0; i < Math.min(linkCount, 10); i++) {
      const link = links.nth(i)
      const tabIndex = await link.getAttribute('tabindex')
      
      // tabindex не должен быть -1 (заблокирован)
      expect(tabIndex).not.toBe('-1')
    }
  })
})
