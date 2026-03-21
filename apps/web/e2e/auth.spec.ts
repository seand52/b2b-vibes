import { test, expect } from './fixtures/auth.fixture';
import { mockProductsAPI } from './utils/api-mocks';

test.describe('Authentication', () => {
  test('redirects unauthenticated users to login', async ({ page }) => {
    // Use raw page without auth mocking - no clientPage fixture
    // Navigate to protected route
    await page.goto('http://localhost:3000/products');

    // Wait for navigation to complete
    await page.waitForLoadState('networkidle');

    // Expect redirect to Auth0 login endpoint
    // Auth0 middleware should redirect to /api/auth/login
    const currentUrl = page.url();
    expect(currentUrl).toContain('/api/auth/login');
  });

  test('authenticated user can access protected routes', async ({ clientPage }) => {
    // Use clientPage fixture which has Auth0 session mocked
    // Mock the products API to return empty array
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
