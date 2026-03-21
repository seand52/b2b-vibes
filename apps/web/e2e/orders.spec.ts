import { test, expect } from './fixtures/auth.fixture';
import {
  mockOrdersAPI,
  mockOrderDetailAPI,
  mockOrderCancelAPI,
  createMockOrder,
} from './utils/api-mocks';

test.describe('Order Management', () => {
  test('view order history and order details', async ({ clientPage }) => {
    const orders = [
      createMockOrder({
        id: '11111111-1111-1111-1111-111111111111',
        status: 'pending',
        item_count: 2,
        total_quantity: 5,
        created_at: '2025-01-15T10:30:00Z',
      }),
      createMockOrder({
        id: '22222222-2222-2222-2222-222222222222',
        status: 'approved',
        item_count: 1,
        total_quantity: 3,
        created_at: '2025-01-10T14:20:00Z',
      }),
      createMockOrder({
        id: '33333333-3333-3333-3333-333333333333',
        status: 'delivered',
        item_count: 3,
        total_quantity: 10,
        created_at: '2025-01-05T09:15:00Z',
      }),
    ];

    await mockOrdersAPI(clientPage, orders);
    await mockOrderDetailAPI(clientPage, orders[0]);

    await clientPage.goto('/orders');

    // Verify page title
    await expect(clientPage.locator('h1')).toContainText('Orders');

    // Verify all orders are listed
    await expect(clientPage.getByText('Order #11111111')).toBeVisible();
    await expect(clientPage.getByText('Order #22222222')).toBeVisible();
    await expect(clientPage.getByText('Order #33333333')).toBeVisible();

    // Verify order counts and dates are displayed
    await expect(clientPage.getByText('2 items')).toBeVisible();
    await expect(clientPage.getByText('Jan 15, 2025')).toBeVisible();

    // Click on first order
    await clientPage.getByText('Order #11111111').click();

    // Verify order detail page
    await expect(clientPage).toHaveURL(/\/orders\/11111111-1111-1111-1111-111111111111/);
    await expect(clientPage.locator('h1')).toContainText('Order #11111111');
    await expect(clientPage.getByText('Pending')).toBeVisible();
    await expect(clientPage.getByText('Order Items')).toBeVisible();
  });

  test('cancel a pending order', async ({ clientPage }) => {
    const pendingOrder = createMockOrder({
      id: '44444444-4444-4444-4444-444444444444',
      status: 'pending',
      item_count: 2,
      total_quantity: 5,
      created_at: '2025-01-20T12:00:00Z',
    });

    const cancelledOrder = {
      ...pendingOrder,
      status: 'cancelled' as const,
    };

    await mockOrdersAPI(clientPage, [pendingOrder]);
    await mockOrderDetailAPI(clientPage, pendingOrder);
    await mockOrderCancelAPI(clientPage, pendingOrder.id, cancelledOrder);

    // Navigate to order detail page
    await clientPage.goto(`/orders/${pendingOrder.id}`);

    // Verify page loaded
    await expect(clientPage.locator('h1')).toContainText('Order #44444444');
    await expect(clientPage.getByText('Pending')).toBeVisible();

    // Click cancel button to open dialog
    await clientPage.getByRole('button', { name: /cancel order/i }).click();

    // Verify confirmation dialog appears
    await expect(
      clientPage.getByText('Cancel this order?')
    ).toBeVisible();
    await expect(
      clientPage.getByText('This action cannot be undone')
    ).toBeVisible();

    // Confirm cancellation by clicking the button in the dialog footer
    await clientPage.getByRole('button', { name: /cancel order/i }).last().click();

    // Verify redirect to orders list
    await expect(clientPage).toHaveURL('/orders');

    // Verify success message (toast notification)
    await expect(clientPage.getByText('Order cancelled')).toBeVisible();
  });
});
