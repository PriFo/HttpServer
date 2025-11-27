import { NextResponse } from 'next/server'

export async function GET() {
  return NextResponse.json({
    success: true,
    message: 'Next.js API route is working',
    timestamp: new Date().toISOString(),
    nodeEnv: process.env.NODE_ENV,
  })
}

