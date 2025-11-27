import { NextRequest, NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/api-config';

const API_BASE_URL = getBackendUrl();

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const resolvedParams = await params
    const { clientId, projectId } = resolvedParams

    // Валидация параметров
    if (!clientId || !projectId) {
      console.error('Missing clientId or projectId:', { clientId, projectId })
      return NextResponse.json(
        { error: 'Missing clientId or projectId' },
        { status: 400 }
      )
    }

    // Проверяем, что это числа
    const clientIdNum = parseInt(clientId, 10)
    const projectIdNum = parseInt(projectId, 10)
    
    if (isNaN(clientIdNum) || isNaN(projectIdNum)) {
      console.error('Invalid clientId or projectId:', { clientId, projectId })
      return NextResponse.json(
        { error: 'Invalid clientId or projectId' },
        { status: 400 }
      )
    }

    const { searchParams } = new URL(request.url);
    
    const dbId = searchParams.get('db_id');
    const page = searchParams.get('page') || '1';
    const limit = searchParams.get('limit') || '20';

    if (!dbId) {
      return NextResponse.json(
        { error: 'db_id parameter is required' },
        { status: 400 }
      );
    }

    const url = new URL(`${API_BASE_URL}/api/clients/${clientIdNum}/projects/${projectIdNum}/normalization/groups`);
    url.searchParams.set('db_id', dbId);
    url.searchParams.set('page', page);
    url.searchParams.set('limit', limit);

    const response = await fetch(url.toString(), {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
      signal: AbortSignal.timeout(10000), // 10 секунд таймаут
    });

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error')
      let errorData: { error?: string } = {}
      try {
        errorData = JSON.parse(errorText)
      } catch {
        // Если не JSON, используем текст как есть
      }
      
      console.error(`Backend error (${response.status}):`, errorText)
      return NextResponse.json(
        { error: errorData.error || errorText || 'Failed to fetch normalization groups' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching normalization groups:', error);
    const errorMessage = error instanceof Error ? error.message : 'Unknown error'
    return NextResponse.json(
      { error: `Internal server error: ${errorMessage}` },
      { status: 500 }
    );
  }
}

