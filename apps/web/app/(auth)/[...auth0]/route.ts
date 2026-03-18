import { auth0 } from '@/lib/auth/config';
import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest): Promise<NextResponse> {
  return auth0.middleware(request);
}

export async function POST(request: NextRequest): Promise<NextResponse> {
  return auth0.middleware(request);
}
