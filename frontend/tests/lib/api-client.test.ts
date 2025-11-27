import { describe, it, expect, vi, afterEach } from 'vitest'
import { apiClient } from '@/lib/api-client'

const originalFetch = global.fetch

function createAbortError(message: string) {
  const error = new Error(message)
  error.name = 'AbortError'
  return error
}

describe('apiClient timeout handling', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
    global.fetch = originalFetch
  })

  it('aborts request after the provided timeoutMs', async () => {
    vi.useFakeTimers()

    const fetchMock = vi.spyOn(global, 'fetch').mockImplementation((_url, init?: RequestInit) => {
      const signal = init?.signal as AbortSignal | undefined
      return new Promise((_resolve, reject) => {
        signal?.addEventListener('abort', () => {
          reject(createAbortError('Request aborted by timeout'))
        })
      })
    })

    const promise = apiClient('/api/test-timeout', { timeoutMs: 500 })

    await vi.advanceTimersByTimeAsync(500)

    await expect(promise).rejects.toThrow('Превышено время ожидания ответа от сервера')
    expect(fetchMock).toHaveBeenCalledWith('/api/test-timeout', expect.objectContaining({ signal: expect.any(AbortSignal) }))
  })

  it('respects external signal when timeout is disabled', async () => {
    vi.useFakeTimers()

    let capturedSignal: AbortSignal | undefined
    vi.spyOn(global, 'fetch').mockImplementation((_url, init?: RequestInit) => {
      capturedSignal = init?.signal as AbortSignal | undefined
      return new Promise((_resolve, reject) => {
        capturedSignal?.addEventListener('abort', () => {
          reject(createAbortError('External abort'))
        })
      })
    })

    const externalController = new AbortController()
    const promise = apiClient('/api/external-signal', {
      timeoutMs: 0,
      signal: externalController.signal,
    })

    await vi.advanceTimersByTimeAsync(5000)

    expect(capturedSignal).toBe(externalController.signal)

    externalController.abort()

    await expect(promise).rejects.toThrow('Запрос был прерван')
  })
})

