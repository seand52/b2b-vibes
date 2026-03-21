import { test, expect } from './fixtures/auth.fixture';
import { mockProductsAPI } from './utils/api-mocks';

test.describe('Authentication', () => {
  test('redirects unauthenticated users to login', async ({ page }) => {
    // Use raw page without auth mocking - no clientPage fixture
    // Navigate to protected route
    await page.goto('http://localhost:3000/products');

    // Wait for navigation to complete
    await page.waitForLoadState('networkidle');

    // Expect redirect to Auth0 login page
    // Auth0 middleware redirects to Auth0's Universal Login
    const currentUrl = page.url();
    expect(currentUrl).toMatch(/auth0\.com|\/auth\/login/);
  });

  test('authenticated user can access protected routes', async ({ clientPage }) => {
    // clientPage has real Auth0 session from global setup
    // Mock the products API to return empty array (backend not running)
    await mockProductsAPI(clientPage, []);

    // Navigate to protected route
    await clientPage.goto('http://localhost:3000/products');

    // Wait for page to load
    await clientPage.waitForLoadState('networkidle');

    // Should successfully load products page
    const currentUrl = clientPage.url();
    expect(currentUrl).toBe('http://localhost:3000/products');

    // Verify page loaded by checking for products heading
    const heading = clientPage.getByRole('heading', { name: /products/i });
    await expect(heading).toBeVisible();
  });
});
