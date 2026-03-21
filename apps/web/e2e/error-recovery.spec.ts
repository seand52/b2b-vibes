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
});
