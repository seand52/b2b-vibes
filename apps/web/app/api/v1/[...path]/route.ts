import { NextRequest, NextResponse } from 'next/server';
import { auth0 } from '@/lib/auth/config';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080';

/**
 * Proxy API route that forwards requests to the Go backend with JWT auth
 */
async function proxyRequest(request: NextRequest) {
  try {
    // Get the session
    const session = await auth0.getSession();
    if (!session?.tokenSet?.accessToken) {
      return NextResponse.json(
        { error: 'unauthorized', message: 'Not authenticated' },
        { status: 401 }
      );
    }

    // Get the path from the URL
    const url = new URL(request.url);
    const path = url.pathname.replace('/api', '');
    const queryString = url.search;

    // Forward the request
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${session.tokenSet.accessToken}`,
    };

    // Copy relevant headers
    const requestId = request.headers.get('X-Request-ID');
    if (requestId) {
      headers['X-Request-ID'] = requestId;
    }

    const fetchOptions: RequestInit = {
      method: request.method,
      headers,
    };

    // Include body for non-GET requests
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      try {
        const body = await request.json();
        fetchOptions.body = JSON.stringify(body);
      } catch {
        // No body or invalid JSON - that's fine for some requests
      }
    }

    const response = await fetch(
      `${API_BASE_URL}/api${path}${queryString}`,
      fetchOptions
    );

    // Handle 204 No Content
    if (response.status === 204) {
      return new NextResponse(null, { status: 204 });
    }

    // Forward response
    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error) {
    console.error('API proxy error:', error);
    return NextResponse.json(
      { error: 'internal_error', message: 'Failed to proxy request' },
      { status: 500 }
    );
  }
}

export async function GET(request: NextRequest) {
  return proxyRequest(request);
}

export async function POST(request: NextRequest) {
  return proxyRequest(request);
}

export async function PUT(request: NextRequest) {
  return proxyRequest(request);
}

export async function DELETE(request: NextRequest) {
  return proxyRequest(request);
}

export async function PATCH(request: NextRequest) {
  return proxyRequest(request);
}
