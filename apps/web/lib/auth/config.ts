import { Auth0Client } from '@auth0/nextjs-auth0/server';

export const auth0 = new Auth0Client({
  domain: process.env.AUTH0_DOMAIN!,
  clientId: process.env.AUTH0_CLIENT_ID!,
  clientSecret: process.env.AUTH0_CLIENT_SECRET!,
  secret: process.env.AUTH0_SECRET!,
  appBaseUrl: process.env.APP_BASE_URL || 'http://localhost:3000',
  authorizationParameters: {
    audience: process.env.AUTH0_AUDIENCE,
    scope: 'openid profile email',
    prompt: 'login', // Always show login screen, don't auto-select user
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
 * Decode JWT payload (without verification - backend handles that)
 */
function decodeJwtPayload(token: string): Record<string, unknown> {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return {};

    const payload = parts[1];
    const decoded = Buffer.from(payload, 'base64url').toString('utf-8');
    return JSON.parse(decoded);
  } catch {
    return {};
  }
}

/**
 * Extract roles from access token
 * The ROLE_CLAIM should be a custom namespace claim (e.g., https://your-domain.com/roles)
 */
export function extractRolesFromAccessToken(accessToken: string): string[] {
  if (!ROLE_CLAIM || !accessToken) {
    return [];
  }

  const claims = decodeJwtPayload(accessToken);
  const roles = claims[ROLE_CLAIM];

  if (Array.isArray(roles)) {
    return roles.filter((r): r is string => typeof r === 'string');
  }
  return [];
}
