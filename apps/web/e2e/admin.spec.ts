import { test, expect } from './fixtures/auth.fixture';
import {
  mockAdminOrdersAPI,
  mockAdminOrderDetailAPI,
  mockAdminApproveOrderAPI,
  mockAdminRejectOrderAPI,
  createMockOrder,
} from './utils/api-mocks';

test.describe('Admin Order Workflow', () => {
  test('admin approves a pending order', async ({ adminPage }) => {
    const order = createMockOrder({ status: 'pending' });
    const approvedOrder = { ...order, status: 'approved' as const };

    await mockAdminOrdersAPI(adminPage, [order]);
    await mockAdminOrderDetailAPI(adminPage, order);
    await mockAdminApproveOrderAPI(adminPage, order.id, approvedOrder);

    await adminPage.goto('http://localhost:3000/admin/orders');
    await adminPage.waitForLoadState('networkidle');

    // Click on the order row by clicking the chevron link
    await adminPage.locator(`a[href="/admin/orders/${order.id}"]`).click();
    await adminPage.waitForLoadState('networkidle');

    // Click approve button
    await adminPage.locator('button', { hasText: 'Approve' }).click();

    // Verify toast and redirect
    await expect(adminPage.locator('[data-sonner-toast]')).toContainText('approved');
    await expect(adminPage).toHaveURL('http://localhost:3000/admin/orders');
  });

  test('admin rejects an order with reason', async ({ adminPage }) => {
    const order = createMockOrder({ status: 'pending' });
    const rejectedOrder = { ...order, status: 'rejected' as const, rejection_reason: 'Out of stock' };

    await mockAdminOrdersAPI(adminPage, [order]);
    await mockAdminOrderDetailAPI(adminPage, order);
    await mockAdminRejectOrderAPI(adminPage, order.id, rejectedOrder);

    await adminPage.goto('http://localhost:3000/admin/orders');
    await adminPage.waitForLoadState('networkidle');

    // Click on the order row
    await adminPage.locator(`a[href="/admin/orders/${order.id}"]`).click();
    await adminPage.waitForLoadState('networkidle');

    // Click reject button to open dialog
    await adminPage.locator('button', { hasText: 'Reject' }).click();

    // Fill in rejection reason in the dialog
    await adminPage.locator('#reason').fill('Out of stock');

    // Click the "Reject Order" button in the dialog footer
    await adminPage.locator('button', { hasText: 'Reject Order' }).click();

    // Verify toast and redirect
    await expect(adminPage.locator('[data-sonner-toast]')).toContainText('rejected');
    await expect(adminPage).toHaveURL('http://localhost:3000/admin/orders');
  });

  test('non-admin user cannot access admin routes', async ({ clientPage }) => {
    // Try to navigate to admin route
    await clientPage.goto('http://localhost:3000/admin/orders');
    await clientPage.waitForLoadState('networkidle');

    // Verify user is redirected away from admin or sees access denied
    // The actual behavior depends on the middleware implementation
    // Common patterns: redirect to home, show 403, or redirect to login
    const currentUrl = clientPage.url();

    // Should not be on the admin orders page
    expect(currentUrl).not.toBe('http://localhost:3000/admin/orders');

    // Common redirect targets - check for at least one
    const isRedirectedAway =
      currentUrl === 'http://localhost:3000/' ||
      currentUrl.includes('/unauthorized') ||
      currentUrl.includes('/403') ||
      currentUrl.includes('/api/auth/login');

    expect(isRedirectedAway).toBe(true);
  });
});
