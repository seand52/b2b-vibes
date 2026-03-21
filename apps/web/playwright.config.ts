import { defineConfig, devices } from '@playwright/test';

const isCI = !!process.env.CI;

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

  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'off',
    actionTimeout: 10000,
    navigationTimeout: 30000,
  },

  projects: [
    // Client user tests
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // Use system chromium on NixOS via PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH env var
        launchOptions: {
          executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
        },
      },
    },

    // Admin tests (same browser, different test files)
    {
      name: 'chromium-admin',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
        },
      },
      testMatch: /admin\.spec\.ts/,
    },
  ],

  // Output directories
  outputDir: 'e2e/test-results',
});
