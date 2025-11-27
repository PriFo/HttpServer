import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleFetchError } from '@/lib/error-handler'
import { NotFoundError, BackendResponseError } from '@/lib/errors'

export const runtime = 'nodejs'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ filename: string }> }
) {
  const startTime = Date.now()
  const { filename } = await params
  const context = createApiContext('/api/databases/backups/[filename]', 'GET', { filename })

  return withLogging(
    'GET /api/databases/backups/[filename]',
    async () => {
      if (!filename) {
        throw new NotFoundError('Backup file', { filename })
      }

      const backendUrl = getBackendUrl()
      const endpoint = `${backendUrl}/api/databases/backups/${encodeURIComponent(filename)}`

      logger.logRequest('GET', `/api/databases/backups/${filename}`, context)

      try {
        const response = await fetch(endpoint, {
          method: 'GET',
          headers: {
            'Accept': 'application/octet-stream',
          },
          cache: 'no-store',
        })

        const duration = Date.now() - startTime
        logger.logResponse('GET', `/api/databases/backups/${filename}`, response.status, duration, {
          ...context,
          fileSize: response.headers.get('Content-Length'),
        })

        if (!response.ok) {
          const errorText = await response.text().catch(() => '')
          throw new BackendResponseError(
            'Failed to download backup',
            response.status,
            { errorText, filename, endpoint }
          )
        }

        // Получаем blob для скачивания
        const blob = await response.blob()
        
        // Определяем Content-Type из ответа бэкенда или используем по умолчанию
        const contentType = response.headers.get('Content-Type') || 'application/octet-stream'
        const contentDisposition = response.headers.get('Content-Disposition') || `attachment; filename="${filename}"`

        logger.info(`Successfully downloaded backup: ${filename}`, {
          ...context,
          size: blob.size,
          contentType,
        })

        return new NextResponse(blob, {
          headers: {
            'Content-Type': contentType,
            'Content-Disposition': contentDisposition,
            'Content-Length': blob.size.toString(),
          },
        })
      } catch (error) {
        const duration = Date.now() - startTime
        return handleFetchError(error, endpoint, { ...context, duration })
      }
    },
    context
  )
}

