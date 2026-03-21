import type { Page } from '@playwright/test';
import type {
  Product,
  CartResponse,
  CartItemResponse,
  Order,
  OrderStatus,
} from '../../lib/types';

// Mock data generators
export function createMockProduct(overrides: Partial<Product> = {}): Product {
  const id = overrides.id || crypto.randomUUID();
  return {
    id,
    sku: `SKU-${id.slice(0, 8)}`,
    name: 'Test Product',
    description: 'A test product description',
    category: 'Test Category',
    price: 99.99,
    tax_rate: 21,
    stock_quantity: 100,
    min_order_quantity: 1,
    images: [],
    ...overrides,
  };
}

export function createMockCartItem(
  product: Product,
  quantity: number = 1
): CartItemResponse {
  return {
    product_id: product.id,
    product_name: product.name,
    product_sku: product.sku,
    quantity,
    unit_price: product.price,
    line_total: product.price * quantity,
    stock_available: product.stock_quantity,
    min_order_quantity: product.min_order_quantity,
    in_stock: product.stock_quantity > 0,
  };
}

export function createMockCart(
  items: CartItemResponse[] = [],
  overrides: Partial<CartResponse> = {}
): CartResponse {
  const subtotal = items.reduce((sum, item) => sum + item.line_total, 0);
  const taxRate = 21;
  const taxAmount = subtotal * (taxRate / 100);

  return {
    id: crypto.randomUUID(),
    status: 'draft',
    items,
    summary: {
      subtotal,
      tax_rate: taxRate,
      tax_amount: taxAmount,
      total: subtotal + taxAmount,
      item_count: items.length,
      total_units: items.reduce((sum, item) => sum + item.quantity, 0),
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

export function createMockOrder(
  overrides: Partial<Order> = {}
): Order {
  const id = overrides.id || crypto.randomUUID();
  return {
    id,
    status: 'pending' as OrderStatus,
    notes: undefined,
    created_at: new Date().toISOString(),
    items: [{ product_id: crypto.randomUUID(), quantity: 2 }],
    item_count: 1,
    total_quantity: 2,
    ...overrides,
  };
}

// API route interceptors
export async function mockProductsAPI(
  page: Page,
  products: Product[]
): Promise<void> {
  await page.route('**/api/v1/products*', async (route) => {
    const url = new URL(route.request().url());
    const search = url.searchParams.get('search')?.toLowerCase();

    let filteredProducts = products;
    if (search) {
      filteredProducts = products.filter(
        (p) =>
          p.name.toLowerCase().includes(search) ||
          p.sku.toLowerCase().includes(search)
      );
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(filteredProducts),
    });
  });
}

export async function mockCartAPI(
  page: Page,
  cart: CartResponse | null
): Promise<void> {
  await page.route('**/api/v1/cart', async (route) => {
    const method = route.request().method();

    if (method === 'GET') {
      if (cart === null) {
        await route.fulfill({ status: 404 });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(cart),
      });
    } else if (method === 'POST') {
      // Add to cart or create cart
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(cart),
      });
    } else if (method === 'DELETE') {
      await route.fulfill({ status: 204 });
    } else {
      await route.continue();
    }
  });
}

export async function mockCartItemsAPI(
  page: Page,
  cart: CartResponse
): Promise<void> {
  // Handle add/update/delete cart items
  await page.route('**/api/v1/cart/items*', async (route) => {
    const method = route.request().method();

    if (method === 'POST' || method === 'PUT' || method === 'PATCH') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(cart),
      });
    } else if (method === 'DELETE') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(cart),
      });
    } else {
      await route.continue();
    }
  });
}

export async function mockCartSubmitAPI(
  page: Page,
  order: Order
): Promise<void> {
  await page.route('**/api/v1/cart/submit', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(order),
    });
  });
}

export async function mockOrdersAPI(
  page: Page,
  orders: Order[]
): Promise<void> {
  await page.route('**/api/v1/orders', async (route) => {
    const url = new URL(route.request().url());
    const statusFilter = url.searchParams.get('status');

    let filteredOrders = orders;
    if (statusFilter && statusFilter !== 'all') {
      filteredOrders = orders.filter((o) => o.status === statusFilter);
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(filteredOrders),
    });
  });
}

export async function mockOrderDetailAPI(
  page: Page,
  order: Order
): Promise<void> {
  await page.route(`**/api/v1/orders/${order.id}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(order),
    });
  });
}

export async function mockOrderCancelAPI(
  page: Page,
  orderId: string,
  updatedOrder: Order
): Promise<void> {
  await page.route(`**/api/v1/orders/${orderId}/cancel`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(updatedOrder),
    });
  });
}

export async function mockAdminOrdersAPI(
  page: Page,
  orders: Order[]
): Promise<void> {
  await page.route('**/api/v1/admin/orders*', async (route) => {
    const url = new URL(route.request().url());
    const statusFilter = url.searchParams.get('status');

    let filteredOrders = orders;
    if (statusFilter && statusFilter !== 'all') {
      filteredOrders = orders.filter((o) => o.status === statusFilter);
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(filteredOrders),
    });
  });
}

export async function mockAdminOrderDetailAPI(
  page: Page,
  order: Order
): Promise<void> {
  await page.route(`**/api/v1/admin/orders/${order.id}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(order),
    });
  });
}

export async function mockAdminApproveOrderAPI(
  page: Page,
  orderId: string,
  approvedOrder: Order
): Promise<void> {
  await page.route(`**/api/v1/admin/orders/${orderId}/approve`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(approvedOrder),
    });
  });
}

export async function mockAdminRejectOrderAPI(
  page: Page,
  orderId: string,
  rejectedOrder: Order
): Promise<void> {
  await page.route(`**/api/v1/admin/orders/${orderId}/reject`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(rejectedOrder),
    });
  });
}

// Error simulation
export async function simulateNetworkError(
  page: Page,
  urlPattern: string
): Promise<void> {
  await page.route(urlPattern, (route) => route.abort('failed'));
}

export async function simulateServerError(
  page: Page,
  urlPattern: string
): Promise<void> {
  await page.route(urlPattern, async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'Internal server error' }),
    });
  });
}
