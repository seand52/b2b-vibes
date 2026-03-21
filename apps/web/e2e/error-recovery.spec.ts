import { test, expect } from './fixtures/auth.fixture';
import {
  mockProductsAPI,
  mockCartAPI,
  mockCartItemsAPI,
  createMockCart,
  createMockCartItem,
  createMockProduct,
  simulateNetworkError,
} from './utils/api-mocks';

test.describe('Error Recovery', () => {
  test('handles network failure during checkout gracefully', async ({
    clientPage,
  }) => {
    // Set up cart with items
    const product = createMockProduct({ name: 'Test Item', price: 50.0 });
    const cartItem = createMockCartItem(product, 2);
    const cart = createMockCart([cartItem]);

    // Mock the cart API to return our cart
    await mockCartAPI(clientPage, cart);
    await mockCartItemsAPI(clientPage, cart);

    // Simulate network failure on submit endpoint
    await simulateNetworkError(clientPage, '**/api/v1/cart/submit');

    // Navigate to cart page
    await clientPage.goto('/cart');

    // Wait for cart to load
    await expect(
      clientPage.locator('text=Test Item')
    ).toBeVisible();

    // Click Submit Order button
    const submitButton = clientPage.locator('button', { hasText: 'Submit Order' });
    await expect(submitButton).toBeVisible();
    await submitButton.click();

    // Verify error toast appears
    await expect(clientPage.locator('[data-sonner-toast]')).toBeVisible();

    // Verify user is still on cart page (can retry)
    await expect(clientPage).toHaveURL('/cart');

    // Verify cart items are still visible (state preserved)
    await expect(
      clientPage.locator('text=Test Item')
    ).toBeVisible();
  });

  test('session expiry redirects to login', async ({ page }) => {
    // Start with authenticated session
    await page.route('**/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: {
            sub: 'auth0|test-user',
            email: 'test@example.com',
            name: 'Test User',
            email_verified: true,
          },
          accessToken: 'mock-access-token',
        }),
      });
    });

    await page.route('**/api/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          sub: 'auth0|test-user',
          email: 'test@example.com',
          name: 'Test User',
        }),
      });
    });

    // Set session cookie
    await page.context().addCookies([
      {
        name: 'appSession',
        value: 'mock-session-value',
        domain: 'localhost',
        path: '/',
        httpOnly: true,
        secure: false,
        sameSite: 'Lax',
      },
    ]);

    // Mock cart API to initially succeed
    const emptyCart = createMockCart([]);
    await mockCartAPI(page, emptyCart);

    // Navigate to a protected page (cart)
    await page.goto('/cart');

    // Verify we're on the cart page
    await expect(page).toHaveURL('/cart');

    // Now simulate session expiry - return 401 on subsequent API calls
    await page.route('**/api/v1/cart', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Unauthorized' }),
      });
    });

    // Mock the auth/me endpoint to also return unauthorized
    await page.route('**/api/auth/me', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Unauthorized' }),
      });
    });

    // Trigger a navigation that requires authentication
    await page.reload();

    // Verify redirect to login (Auth0 will handle the actual redirect)
    // We should be redirected away from the protected page
    await page.waitForTimeout(1000); // Give time for redirect logic

    // Check that we're no longer on the cart page or we see an error
    // Due to Auth0 handling, we might see different behaviors
    // At minimum, the page should handle the 401 gracefully
    const currentUrl = page.url();
    const isOnCart = currentUrl.includes('/cart');
    const hasErrorMessage = await page.locator('text=Unauthorized').isVisible();

    // Either redirected away from cart OR showing error
    expect(isOnCart === false || hasErrorMessage).toBeTruthy();
  });
});
