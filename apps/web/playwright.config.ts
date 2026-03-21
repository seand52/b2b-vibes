import { defineConfig, devices } from '@playwright/test';

const isCI = !!process.env.CI;

// Common launch options for NixOS compatibility
const launchOptions = {
  executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
};

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: isCI,
  retries: isCI ? 2 : 0,
  workers: isCI ? 1 : undefined,
  reporter: [
    ['html', { open: 'never' }],
    ['list'],
    ...(isCI ? [['github'] as const] : []),
  ],

  // Global setup - runs once before all tests to authenticate
  globalSetup: './e2e/global.setup.ts',

  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'off',
    actionTimeout: 10000,
    navigationTimeout: 30000,
  },

  projects: [
    // Client user tests (most tests)
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions,
      },
      // Exclude admin-specific tests
      testIgnore: /admin\.spec\.ts/,
    },

    // Admin tests
    {
      name: 'chromium-admin',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions,
      },
      testMatch: /admin\.spec\.ts/,
    },
  ],

  // Output directories
  outputDir: 'e2e/test-results',
});
