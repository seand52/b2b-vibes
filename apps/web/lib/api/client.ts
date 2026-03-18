import { auth0 } from '@/lib/auth/config';
import type { APIError } from '@/lib/types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export class ApiClientError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}

interface FetchOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
}

/**
 * Server-side API client that forwards JWT tokens to the Go backend
 */
export async function apiClient<T>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<T> {
  const { body, headers: customHeaders, ...rest } = options;

  // Get access token from Auth0 session
  const session = await auth0.getSession();
  if (!session?.tokenSet?.accessToken) {
    throw new ApiClientError('Not authenticated', 401);
  }

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${session.tokenSet.accessToken}`,
    ...customHeaders,
  };

  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...rest,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  const data = await response.json();

  if (!response.ok) {
    const error = data as APIError;
    throw new ApiClientError(
      error.message || error.error || 'An error occurred',
      response.status,
      error.code
    );
  }

  return data as T;
}

/**
 * Client-side API wrapper that calls our Next.js API routes
 * which in turn call the Go backend with proper auth
 */
export async function clientApiClient<T>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<T> {
  const { body, headers: customHeaders, ...rest } = options;

  const response = await fetch(`/api${endpoint}`, {
    ...rest,
    headers: {
      'Content-Type': 'application/json',
      ...customHeaders,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  const data = await response.json();

  if (!response.ok) {
    const error = data as APIError;
    throw new ApiClientError(
      error.message || error.error || 'An error occurred',
      response.status,
      error.code
    );
  }

  return data as T;
}
