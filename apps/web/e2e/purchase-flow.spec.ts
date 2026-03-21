import { test, expect } from './fixtures/auth.fixture';
import {
  mockProductsAPI,
  createMockProduct,
  mockCartAPI,
  mockCartItemsAPI,
  mockCartSubmitAPI,
  mockOrderDetailAPI,
  createMockCart,
  createMockCartItem,
  createMockOrder,
} from './utils/api-mocks';

test.describe('Purchase Flow', () => {
  test('complete purchase flow: browse, add to cart, submit order', async ({
    clientPage,
  }) => {
    // Setup mock products
    const productA = createMockProduct({
      name: 'Widget A',
      price: 29.99,
      stock_quantity: 50,
      min_order_quantity: 1,
    });
    const productB = createMockProduct({
      name: 'Widget B',
      price: 49.99,
      stock_quantity: 30,
      min_order_quantity: 1,
    });
    const products = [productA, productB];

    await mockProductsAPI(clientPage, products);

    // Initially no cart exists
    await mockCartAPI(clientPage, null);

    // Navigate to products page
    await clientPage.goto('http://localhost:3000/products');
    await expect(clientPage.locator('h1')).toContainText('Products');

    // Verify products are displayed
    await expect(clientPage.getByText('Widget A')).toBeVisible();
    await expect(clientPage.getByText('Widget B')).toBeVisible();

    // Setup cart mock for after adding item
    const cartItem = createMockCartItem(productA, 1);
    const cart = createMockCart([cartItem]);
    await mockCartAPI(clientPage, cart);
    await mockCartItemsAPI(clientPage, cart);

    // Add first product to cart
    const addToCartButtons = clientPage.getByRole('button', {
      name: /add to cart/i,
    });
    await addToCartButtons.first().click();

    // Wait for success toast
    await expect(clientPage.getByText(/added.*to cart/i)).toBeVisible();

    // Navigate to cart
    await clientPage.goto('http://localhost:3000/cart');
    await expect(clientPage.locator('h1')).toContainText('Your Cart');

    // Verify cart contains the item
    await expect(clientPage.getByText('Widget A')).toBeVisible();
    await expect(clientPage.getByText('€29.99 each')).toBeVisible();

    // Setup order mock for submission
    const order = createMockOrder({
      items: [{ product_id: productA.id, quantity: 1 }],
      item_count: 1,
      total_quantity: 1,
    });
    await mockCartSubmitAPI(clientPage, order);
    await mockOrderDetailAPI(clientPage, order);

    // Submit order
    await clientPage.getByRole('button', { name: /submit order/i }).click();

    // Verify redirect to order detail page
    await expect(clientPage).toHaveURL(/\/orders\/.+/);
    await expect(clientPage.locator('h1')).toContainText('Order #');
  });

  test('update cart quantities and submit', async ({ clientPage }) => {
    // Setup mock products
    const product = createMockProduct({
      name: 'Premium Widget',
      price: 99.99,
      stock_quantity: 100,
      min_order_quantity: 1,
    });

    await mockProductsAPI(clientPage, [product]);

    // Start with items already in cart
    const initialItem = createMockCartItem(product, 2);
    const initialCart = createMockCart([initialItem]);
    await mockCartAPI(clientPage, initialCart);

    // Navigate to cart
    await clientPage.goto('http://localhost:3000/cart');
    await expect(clientPage.locator('h1')).toContainText('Your Cart');

    // Verify initial quantity
    await expect(clientPage.getByText('Premium Widget')).toBeVisible();
    const quantityDisplay = clientPage
      .locator('.font-medium')
      .filter({ hasText: /^2$/ });
    await expect(quantityDisplay.first()).toBeVisible();

    // Setup updated cart mock with increased quantity
    const updatedItem = createMockCartItem(product, 3);
    const updatedCart = createMockCart([updatedItem]);
    await mockCartItemsAPI(clientPage, updatedCart);
    await mockCartAPI(clientPage, updatedCart);

    // Increment quantity using + button
    const incrementButton = clientPage
      .getByRole('button', { name: '' })
      .filter({ has: clientPage.locator('svg.lucide-plus') });
    await incrementButton.first().click();

    // Wait for update and verify new quantity
    await clientPage.waitForTimeout(500);
    const newQuantityDisplay = clientPage
      .locator('.font-medium')
      .filter({ hasText: /^3$/ });
    await expect(newQuantityDisplay.first()).toBeVisible();

    // Setup order mock for submission
    const order = createMockOrder({
      items: [{ product_id: product.id, quantity: 3 }],
      item_count: 1,
      total_quantity: 3,
    });
    await mockCartSubmitAPI(clientPage, order);
    await mockOrderDetailAPI(clientPage, order);

    // Submit order
    await clientPage.getByRole('button', { name: /submit order/i }).click();

    // Verify success and redirect
    await expect(clientPage.getByText(/order submitted successfully/i)).toBeVisible();
    await expect(clientPage).toHaveURL(/\/orders\/.+/);
  });

  test('remove items from cart, continue shopping, complete purchase', async ({
    clientPage,
  }) => {
    // Setup mock products
    const productA = createMockProduct({
      name: 'Widget A',
      price: 29.99,
      stock_quantity: 50,
      min_order_quantity: 1,
    });
    const productB = createMockProduct({
      name: 'Widget B',
      price: 49.99,
      stock_quantity: 30,
      min_order_quantity: 1,
    });
    const products = [productA, productB];

    await mockProductsAPI(clientPage, products);

    // Start with two items in cart
    const itemA = createMockCartItem(productA, 2);
    const itemB = createMockCartItem(productB, 1);
    const initialCart = createMockCart([itemA, itemB]);
    await mockCartAPI(clientPage, initialCart);

    // Navigate to cart
    await clientPage.goto('http://localhost:3000/cart');
    await expect(clientPage.locator('h1')).toContainText('Your Cart');

    // Verify both items are present
    await expect(clientPage.getByText('Widget A')).toBeVisible();
    await expect(clientPage.getByText('Widget B')).toBeVisible();

    // Setup cart mock after removing Widget A
    const cartAfterRemoval = createMockCart([itemB]);
    await mockCartItemsAPI(clientPage, cartAfterRemoval);
    await mockCartAPI(clientPage, cartAfterRemoval);

    // Remove Widget A using trash button
    const removeButtons = clientPage
      .getByRole('button', { name: '' })
      .filter({ has: clientPage.locator('svg.lucide-trash-2') });
    await removeButtons.first().click();

    // Wait for success toast
    await expect(clientPage.getByText(/item removed from cart/i)).toBeVisible();

    // Verify Widget A is gone
    await expect(clientPage.getByText('Widget A')).not.toBeVisible();
    await expect(clientPage.getByText('Widget B')).toBeVisible();

    // Click "Continue Shopping" link
    await clientPage.getByRole('link', { name: /continue shopping/i }).click();

    // Verify navigation to products page
    await expect(clientPage).toHaveURL('http://localhost:3000/products');
    await expect(clientPage.locator('h1')).toContainText('Products');

    // Setup cart mock for adding Widget A back
    const newItemA = createMockCartItem(productA, 1);
    const cartWithBoth = createMockCart([itemB, newItemA]);
    await mockCartAPI(clientPage, cartWithBoth);
    await mockCartItemsAPI(clientPage, cartWithBoth);

    // Add Widget A back to cart
    const addToCartButtons = clientPage.getByRole('button', {
      name: /add to cart/i,
    });
    await addToCartButtons.first().click();

    // Wait for success toast
    await expect(clientPage.getByText(/added.*to cart/i)).toBeVisible();

    // Return to cart
    await clientPage.goto('http://localhost:3000/cart');
    await expect(clientPage.locator('h1')).toContainText('Your Cart');

    // Verify both items are now in cart
    await expect(clientPage.getByText('Widget A')).toBeVisible();
    await expect(clientPage.getByText('Widget B')).toBeVisible();

    // Setup order mock for submission
    const order = createMockOrder({
      items: [
        { product_id: productB.id, quantity: 1 },
        { product_id: productA.id, quantity: 1 },
      ],
      item_count: 2,
      total_quantity: 2,
    });
    await mockCartSubmitAPI(clientPage, order);
    await mockOrderDetailAPI(clientPage, order);

    // Submit order
    await clientPage.getByRole('button', { name: /submit order/i }).click();

    // Verify success and redirect
    await expect(clientPage.getByText(/order submitted successfully/i)).toBeVisible();
    await expect(clientPage).toHaveURL(/\/orders\/.+/);
    await expect(clientPage.locator('h1')).toContainText('Order #');
  });
});
