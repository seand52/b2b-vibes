import { test as base, type Page } from '@playwright/test';

// Test user types
export interface TestUser {
  email: string;
  sub: string;
  name: string;
  roles: string[];
}

export const testUsers = {
  client: {
    email: 'test-client@example.com',
    sub: 'auth0|test-client-001',
    name: 'Test Client',
    roles: [],
  },
  admin: {
    email: 'test-admin@example.com',
    sub: 'auth0|test-admin-001',
    name: 'Test Admin',
    roles: ['admin'],
  },
} as const;

// Mock Auth0 session for a user
async function mockAuth0Session(page: Page, user: TestUser): Promise<void> {
  // Intercept the Auth0 /auth/me endpoint that checks session
  await page.route('**/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user: {
          sub: user.sub,
          email: user.email,
          name: user.name,
          email_verified: true,
          [process.env.AUTH0_ROLE_CLAIM || 'https://b2b-orders.com/roles']: user.roles,
        },
        accessToken: 'mock-access-token-for-testing',
      }),
    });
  });

  // Also intercept /api/auth/me for client-side checks
  await page.route('**/api/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        sub: user.sub,
        email: user.email,
        name: user.name,
        [process.env.AUTH0_ROLE_CLAIM || 'https://b2b-orders.com/roles']: user.roles,
      }),
    });
  });

  // Set session cookie to simulate logged-in state
  await page.context().addCookies([
    {
      name: 'appSession',
      value: 'mock-session-value-for-testing',
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      secure: false,
      sameSite: 'Lax',
    },
  ]);
}

// Extended test fixture with auth helpers
export const test = base.extend<{
  clientPage: Page;
  adminPage: Page;
}>({
  // Setup authenticated client session
  clientPage: async ({ page }, use) => {
    await mockAuth0Session(page, testUsers.client);
    await use(page);
  },

  // Setup authenticated admin session
  adminPage: async ({ page }, use) => {
    await mockAuth0Session(page, testUsers.admin);
    await use(page);
  },
});

export { expect } from '@playwright/test';
