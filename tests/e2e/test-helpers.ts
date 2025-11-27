/**
 * Вспомогательные функции для E2E тестов
 * 
 * Общие утилиты, используемые в различных тестах
 */

import { Page, expect } from '@playwright/test'

/**
 * Ожидает появления элемента с несколькими вариантами селекторов
 */
export async function waitForAnyElement(
  page: Page,
  selectors: string[],
  timeout: number = 10000
): Promise<boolean> {
  for (const selector of selectors) {
    try {
      await page.waitForSelector(selector, { timeout: 2000 })
      return true
    } catch {
      continue
    }
  }
  return false
}

/**
 * Проверяет, виден ли хотя бы один из элементов
 */
export async function isAnyVisible(
  page: Page,
  selectors: string[],
  timeout: number = 5000
): Promise<boolean> {
  for (const selector of selectors) {
    try {
      const element = page.locator(selector).first()
      if (await element.isVisible({ timeout: 1000 })) {
        return true
      }
    } catch {
      continue
    }
  }
  return false
}

/**
 * Ожидает завершения загрузки страницы
 */
export async function waitForPageLoad(page: Page): Promise<void> {
  await page.waitForLoadState('networkidle')
  await page.waitForLoadState('domcontentloaded')
  // Дополнительная пауза для завершения всех анимаций
  await page.waitForTimeout(1000)
}

/**
 * Ожидает указанное время (для специфических случаев)
 * Используйте только когда действительно нужна фиксированная задержка
 */
export async function wait(ms: number): Promise<void> {
  await new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * Ожидает появления текста на странице
 */
export async function waitForText(
  page: Page,
  text: string | RegExp,
  timeout: number = 10000
): Promise<void> {
  await expect(page.locator(`text=${text}`).first()).toBeVisible({ timeout })
}

/**
 * Кликает на кнопку, если она видна
 */
export async function clickIfVisible(
  page: Page,
  selectors: string[],
  timeout: number = 5000
): Promise<boolean> {
  for (const selector of selectors) {
    try {
      const element = page.locator(selector).first()
      if (await element.isVisible({ timeout: 2000 })) {
        await element.click()
        await page.waitForTimeout(500)
        return true
      }
    } catch {
      continue
    }
  }
  return false
}

/**
 * Заполняет поле формы, если оно видно
 */
export async function fillIfVisible(
  page: Page,
  selector: string,
  value: string,
  timeout: number = 5000
): Promise<boolean> {
  try {
    const element = page.locator(selector).first()
    if (await element.isVisible({ timeout })) {
      await element.fill(value)
      return true
    }
  } catch {
    return false
  }
  return false
}

/**
 * Ожидает завершения операции с таймаутом
 */
export async function waitForOperation(
  condition: () => Promise<boolean>,
  timeout: number = 30000,
  interval: number = 1000
): Promise<boolean> {
  const startTime = Date.now()
  
  while (Date.now() - startTime < timeout) {
    try {
      const result = await condition()
      if (result) {
        return true
      }
    } catch {
      // Игнорируем ошибки
    }
    
    await new Promise(resolve => setTimeout(resolve, interval))
  }
  
  return false
}

/**
 * Проверяет наличие toast-уведомления
 */
export async function checkToast(
  page: Page,
  text: string | RegExp,
  type: 'success' | 'error' | 'info' | 'warning' = 'success',
  timeout: number = 5000
): Promise<boolean> {
  const toastSelectors = [
    `[role="alert"]:has-text("${text}")`,
    `.toast-${type}:has-text("${text}")`,
    `text=${text}`,
  ]
  
  return await isAnyVisible(page, toastSelectors, timeout)
}

/**
 * Ожидает скачивания файла
 */
export async function waitForDownload(
  page: Page,
  timeout: number = 30000
): Promise<any> {
  const [download] = await Promise.all([
    page.waitForEvent('download', { timeout }).catch(() => null),
  ])
  return download
}

/**
 * Проверяет статус через API
 */
export async function checkApiStatus(
  page: Page,
  endpoint: string,
  expectedStatus: number = 200
): Promise<boolean> {
  try {
    const response = await page.evaluate(async (url) => {
      const res = await fetch(url)
      return { status: res.status, ok: res.ok }
    }, endpoint)
    
    return response.status === expectedStatus
  } catch {
    return false
  }
}

/**
 * Очищает localStorage и sessionStorage
 */
export async function clearStorage(page: Page): Promise<void> {
  await page.evaluate(() => {
    localStorage.clear()
    sessionStorage.clear()
  })
}

/**
 * Устанавливает значение в localStorage
 */
export async function setLocalStorage(
  page: Page,
  key: string,
  value: string
): Promise<void> {
  await page.evaluate(
    ({ k, v }) => {
      localStorage.setItem(k, v)
    },
    { k: key, v: value }
  )
}

/**
 * Получает значение из localStorage
 */
export async function getLocalStorage(
  page: Page,
  key: string
): Promise<string | null> {
  return await page.evaluate((k) => {
    return localStorage.getItem(k)
  }, key)
}

/**
 * Делает скриншот с именем файла
 */
export async function takeScreenshot(
  page: Page,
  name: string
): Promise<void> {
  await page.screenshot({
    path: `test-results/screenshots/${name}-${Date.now()}.png`,
    fullPage: true,
  })
}

/**
 * Логирует информацию о странице
 */
export async function logPageInfo(page: Page): Promise<void> {
  const url = page.url()
  const title = await page.title()
  console.log(`📄 Страница: ${title} (${url})`)
}

/**
 * Ожидает обновления данных через SSE или polling
 * Полезно для проверки обновлений в реальном времени
 */
export async function waitForDataUpdate(
  page: Page,
  selector: string,
  initialValue: string | null,
  timeout: number = 10000,
  interval: number = 1000
): Promise<boolean> {
  const startTime = Date.now()
  
  while (Date.now() - startTime < timeout) {
    try {
      const element = page.locator(selector).first()
      const currentValue = await element.textContent().catch(() => null)
      
      if (currentValue !== initialValue && currentValue !== null) {
        return true
      }
    } catch {
      // Игнорируем ошибки
    }
    
    await new Promise(resolve => setTimeout(resolve, interval))
  }
  
  return false
}
