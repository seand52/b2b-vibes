import { test as base, type Page, type BrowserContext } from '@playwright/test';
import path from 'path';

const AUTH_DIR = path.join(__dirname, '..', '.auth');

// Test user info (for reference in tests)
export const testUsers = {
  client: {
    email: process.env.TEST_CLIENT_EMAIL || 'test-client@example.com',
    name: 'Test Client',
  },
  admin: {
    email: process.env.TEST_ADMIN_EMAIL || 'test-admin@example.com',
    name: 'Test Admin',
  },
} as const;

// Extended test fixture with authenticated pages
export const test = base.extend<{
  clientPage: Page;
  adminPage: Page;
  clientContext: BrowserContext;
  adminContext: BrowserContext;
}>({
  // Create a browser context with client auth state
  clientContext: async ({ browser }, use) => {
    const context = await browser.newContext({
      storageState: path.join(AUTH_DIR, 'client.json'),
    });
    await use(context);
    await context.close();
  },

  // Create a page with client auth state
  clientPage: async ({ clientContext }, use) => {
    const page = await clientContext.newPage();
    await use(page);
  },

  // Create a browser context with admin auth state
  adminContext: async ({ browser }, use) => {
    const context = await browser.newContext({
      storageState: path.join(AUTH_DIR, 'admin.json'),
    });
    await use(context);
    await context.close();
  },

  // Create a page with admin auth state
  adminPage: async ({ adminContext }, use) => {
    const page = await adminContext.newPage();
    await use(page);
  },
});

export { expect } from '@playwright/test';
