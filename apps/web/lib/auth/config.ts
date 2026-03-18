import { Auth0Client } from '@auth0/nextjs-auth0/server';

export const auth0 = new Auth0Client({
  authorizationParameters: {
    audience: process.env.AUTH0_AUDIENCE,
    scope: 'openid profile email',
  },
});

export const ROLE_CLAIM = process.env.AUTH0_ROLE_CLAIM || '';

/**
 * Check if the user has the admin role
 */
export function hasAdminRole(roles: string[]): boolean {
  return roles.includes('admin');
}

/**
 * Extract roles from Auth0 session claims
 * The ROLE_CLAIM should be a custom namespace claim (e.g., https://your-domain.com/roles)
 */
export function extractRoles(claims: Record<string, unknown>): string[] {
  if (!ROLE_CLAIM || !claims[ROLE_CLAIM]) {
    return [];
  }
  const roles = claims[ROLE_CLAIM];
  if (Array.isArray(roles)) {
    return roles.filter((r): r is string => typeof r === 'string');
  }
  return [];
}
