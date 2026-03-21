import { chromium, type FullConfig } from '@playwright/test';
import path from 'path';
import fs from 'fs';
import dotenv from 'dotenv';

// Load test-specific environment variables from .env.test
dotenv.config({ path: path.join(__dirname, '..', '.env.test') });

const AUTH_DIR = path.join(__dirname, '.auth');

// Ensure auth directory exists
if (!fs.existsSync(AUTH_DIR)) {
  fs.mkdirSync(AUTH_DIR, { recursive: true });
}

async function loginViaAuth0(
  email: string,
  password: string,
  storageStatePath: string,
  executablePath?: string
): Promise<void> {
  const browser = await chromium.launch({
    executablePath: executablePath || undefined,
  });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    // Navigate to login
    await page.goto('http://localhost:3000/auth/login');

    // Wait for Auth0 login page (Universal Login)
    await page.waitForURL(/.*auth0.com.*|.*localhost:3000.*/, { timeout: 15000 });

    // Check if we're on Auth0's login page
    if (page.url().includes('auth0.com')) {
      // Fill Auth0 Universal Login form
      // Auth0 Universal Login has different field selectors depending on version
      const emailInput = page.locator('input[name="username"], input[name="email"], input[id="username"]');

      await emailInput.waitFor({ state: 'visible', timeout: 10000 });
      await emailInput.fill(email);

      // Click Continue to proceed to password step (Auth0 Universal Login flow)
      await page.getByRole('button', { name: 'Continue', exact: true }).click();

      // Wait for password field and fill it
      const passwordInput = page.locator('input[name="password"], input[type="password"]');
      await passwordInput.waitFor({ state: 'visible', timeout: 10000 });
      await passwordInput.fill(password);

      // Click Continue to submit login
      await page.getByRole('button', { name: 'Continue', exact: true }).click();

      // Wait for redirect back to app
      await page.waitForURL(/localhost:3000/, { timeout: 30000 });
    }

    // Verify we're logged in by checking we can access a protected route
    await page.goto('http://localhost:3000/products');
    await page.waitForLoadState('networkidle');

    // Should not be redirected to login
    if (page.url().includes('/auth/login') || page.url().includes('auth0.com')) {
      throw new Error(`Login failed for ${email} - still on login page`);
    }

    // Save storage state
    await context.storageState({ path: storageStatePath });
    console.log(`Saved auth state for ${email} to ${storageStatePath}`);
  } finally {
    await browser.close();
  }
}

async function globalSetup(config: FullConfig): Promise<void> {
  const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;

  // Check required environment variables
  const clientEmail = process.env.TEST_CLIENT_EMAIL;
  const clientPassword = process.env.TEST_CLIENT_PASSWORD;
  const adminEmail = process.env.TEST_ADMIN_EMAIL;
  const adminPassword = process.env.TEST_ADMIN_PASSWORD;

  if (!clientEmail || !clientPassword) {
    throw new Error(
      'Missing TEST_CLIENT_EMAIL or TEST_CLIENT_PASSWORD environment variables. ' +
      'Create test users in Auth0 and set these in your .env file.'
    );
  }

  if (!adminEmail || !adminPassword) {
    throw new Error(
      'Missing TEST_ADMIN_EMAIL or TEST_ADMIN_PASSWORD environment variables. ' +
      'Create an admin test user in Auth0 and set these in your .env file.'
    );
  }

  console.log('Setting up authentication...');

  // Login as client user
  console.log(`Logging in as client: ${clientEmail}`);
  await loginViaAuth0(
    clientEmail,
    clientPassword,
    path.join(AUTH_DIR, 'client.json'),
    executablePath
  );

  // Login as admin user
  console.log(`Logging in as admin: ${adminEmail}`);
  await loginViaAuth0(
    adminEmail,
    adminPassword,
    path.join(AUTH_DIR, 'admin.json'),
    executablePath
  );

  console.log('Authentication setup complete!');
}

export default globalSetup;
