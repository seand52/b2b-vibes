'use client';

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from '@tanstack/react-query';
import { clientApiClient, ApiClientError } from './client';
import type {
  Product,
  CartResponse,
  Order,
  Client,
} from '@/lib/types';

// Query Keys
export const queryKeys = {
  products: ['products'] as const,
  product: (id: string) => ['products', id] as const,
  cart: ['cart'] as const,
  orders: ['orders'] as const,
  order: (id: string) => ['orders', id] as const,
  // Admin
  adminOrders: ['admin', 'orders'] as const,
  adminOrder: (id: string) => ['admin', 'orders', id] as const,
  adminClients: ['admin', 'clients'] as const,
  adminClient: (id: string) => ['admin', 'clients', id] as const,
};

// ============ PRODUCTS ============

interface ProductsFilter {
  category?: string;
  search?: string;
}

export function useProducts(
  filter?: ProductsFilter,
  options?: Omit<UseQueryOptions<Product[], ApiClientError>, 'queryKey' | 'queryFn'>
) {
  const params = new URLSearchParams();
  if (filter?.category) params.set('category', filter.category);
  if (filter?.search) params.set('search', filter.search);
  const queryString = params.toString();

  return useQuery({
    queryKey: [...queryKeys.products, filter],
    queryFn: () =>
      clientApiClient<Product[]>(`/v1/products${queryString ? `?${queryString}` : ''}`),
    ...options,
  });
}

export function useProduct(
  id: string,
  options?: Omit<UseQueryOptions<Product, ApiClientError>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: queryKeys.product(id),
    queryFn: () => clientApiClient<Product>(`/v1/products/${id}`),
    enabled: !!id,
    ...options,
  });
}

// ============ CART ============

export function useCart(
  options?: Omit<UseQueryOptions<CartResponse, ApiClientError>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: queryKeys.cart,
    queryFn: () => clientApiClient<CartResponse>('/v1/cart'),
    retry: (failureCount, error) => {
      // Don't retry on 404 (no cart exists)
      if (error.status === 404) return false;
      return failureCount < 3;
    },
    ...options,
  });
}

export function useCreateCart() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () =>
      clientApiClient<CartResponse>('/v1/cart', { method: 'POST' }),
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.cart, data);
    },
  });
}

export function useAddToCart() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ productId, quantity }: { productId: string; quantity: number }) =>
      clientApiClient<CartResponse>('/v1/cart/items', {
        method: 'POST',
        body: { product_id: productId, quantity },
      }),
    onMutate: async () => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: queryKeys.cart });
    },
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.cart, data);
    },
  });
}

export function useUpdateCartQuantity() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ productId, quantity }: { productId: string; quantity: number }) =>
      clientApiClient<CartResponse>(`/v1/cart/items/${productId}`, {
        method: 'PUT',
        body: { quantity },
      }),
    onMutate: async ({ productId, quantity }) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: queryKeys.cart });

      // Snapshot previous value
      const previousCart = queryClient.getQueryData<CartResponse>(queryKeys.cart);

      // Optimistically update cart
      if (previousCart) {
        const updatedCart = {
          ...previousCart,
          items: previousCart.items.map((item) =>
            item.product_id === productId
              ? {
                  ...item,
                  quantity,
                  line_total: item.unit_price * quantity,
                }
              : item
          ),
        };

        // Recalculate summary
        const subtotal = updatedCart.items.reduce((sum, item) => sum + item.line_total, 0);
        const taxAmount = subtotal * (previousCart.summary.tax_rate / 100);
        const total = subtotal + taxAmount;
        const totalUnits = updatedCart.items.reduce((sum, item) => sum + item.quantity, 0);

        updatedCart.summary = {
          ...previousCart.summary,
          subtotal,
          tax_amount: taxAmount,
          total,
          total_units: totalUnits,
        };

        queryClient.setQueryData(queryKeys.cart, updatedCart);
      }

      return { previousCart };
    },
    onError: (_err, _variables, context) => {
      // Rollback on error
      if (context?.previousCart) {
        queryClient.setQueryData(queryKeys.cart, context.previousCart);
      }
    },
    onSuccess: (data) => {
      // Update with server response
      queryClient.setQueryData(queryKeys.cart, data);
    },
  });
}

export function useRemoveFromCart() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (productId: string) =>
      clientApiClient<CartResponse>(`/v1/cart/items/${productId}`, {
        method: 'DELETE',
      }),
    onMutate: async (productId) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: queryKeys.cart });

      // Snapshot previous value
      const previousCart = queryClient.getQueryData<CartResponse>(queryKeys.cart);

      // Optimistically remove item
      if (previousCart) {
        const updatedCart = {
          ...previousCart,
          items: previousCart.items.filter((item) => item.product_id !== productId),
        };

        // Recalculate summary
        const subtotal = updatedCart.items.reduce((sum, item) => sum + item.line_total, 0);
        const taxAmount = subtotal * (previousCart.summary.tax_rate / 100);
        const total = subtotal + taxAmount;
        const totalUnits = updatedCart.items.reduce((sum, item) => sum + item.quantity, 0);

        updatedCart.summary = {
          ...previousCart.summary,
          subtotal,
          tax_amount: taxAmount,
          total,
          item_count: updatedCart.items.length,
          total_units: totalUnits,
        };

        queryClient.setQueryData(queryKeys.cart, updatedCart);
      }

      return { previousCart };
    },
    onError: (_err, _variables, context) => {
      // Rollback on error
      if (context?.previousCart) {
        queryClient.setQueryData(queryKeys.cart, context.previousCart);
      }
    },
    onSuccess: (data) => {
      // Update with server response
      queryClient.setQueryData(queryKeys.cart, data);
    },
  });
}

export function useUpdateCartNotes() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notes: string) =>
      clientApiClient<CartResponse>('/v1/cart/notes', {
        method: 'PUT',
        body: { notes },
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.cart, data);
    },
  });
}

export function useSubmitCart() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () =>
      clientApiClient<Order>('/v1/cart/submit', { method: 'POST' }),
    onSuccess: () => {
      // Invalidate cart and orders
      queryClient.invalidateQueries({ queryKey: queryKeys.cart });
      queryClient.invalidateQueries({ queryKey: queryKeys.orders });
    },
  });
}

export function useDiscardCart() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () =>
      clientApiClient<void>('/v1/cart', { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.setQueryData(queryKeys.cart, null);
      queryClient.invalidateQueries({ queryKey: queryKeys.cart });
    },
  });
}

// ============ ORDERS ============

interface OrdersFilter {
  status?: string;
}

export function useOrders(
  filter?: OrdersFilter,
  options?: Omit<UseQueryOptions<Order[], ApiClientError>, 'queryKey' | 'queryFn'>
) {
  const params = new URLSearchParams();
  if (filter?.status) params.set('status', filter.status);
  const queryString = params.toString();

  return useQuery({
    queryKey: [...queryKeys.orders, filter],
    queryFn: () =>
      clientApiClient<Order[]>(`/v1/orders${queryString ? `?${queryString}` : ''}`),
    ...options,
  });
}

export function useOrder(
  id: string,
  options?: Omit<UseQueryOptions<Order, ApiClientError>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: queryKeys.order(id),
    queryFn: () => clientApiClient<Order>(`/v1/orders/${id}`),
    enabled: !!id,
    ...options,
  });
}

export function useCancelOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (orderId: string) =>
      clientApiClient<void>(`/v1/orders/${orderId}/cancel`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.orders });
    },
  });
}

// ============ ADMIN: ORDERS ============

export function useAdminOrders(
  filter?: OrdersFilter,
  options?: Omit<UseQueryOptions<Order[], ApiClientError>, 'queryKey' | 'queryFn'>
) {
  const params = new URLSearchParams();
  if (filter?.status) params.set('status', filter.status);
  const queryString = params.toString();

  return useQuery({
    queryKey: [...queryKeys.adminOrders, filter],
    queryFn: () =>
      clientApiClient<Order[]>(`/v1/admin/orders${queryString ? `?${queryString}` : ''}`),
    ...options,
  });
}

export function useAdminOrder(
  id: string,
  options?: Omit<UseQueryOptions<Order, ApiClientError>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: queryKeys.adminOrder(id),
    queryFn: () => clientApiClient<Order>(`/v1/admin/orders/${id}`),
    enabled: !!id,
    ...options,
  });
}

export function useApproveOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ orderId, approvedBy }: { orderId: string; approvedBy: string }) =>
      clientApiClient<Order>(`/v1/admin/orders/${orderId}/approve`, {
        method: 'POST',
        body: { approved_by: approvedBy },
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.adminOrders });
    },
  });
}

export function useRejectOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ orderId, reason }: { orderId: string; reason: string }) =>
      clientApiClient<void>(`/v1/admin/orders/${orderId}/reject`, {
        method: 'POST',
        body: { reason },
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.adminOrders });
    },
  });
}

// ============ ADMIN: CLIENTS ============

interface ClientsFilter {
  search?: string;
  active?: boolean;
}

export function useAdminClients(
  filter?: ClientsFilter,
  options?: Omit<UseQueryOptions<Client[], ApiClientError>, 'queryKey' | 'queryFn'>
) {
  const params = new URLSearchParams();
  if (filter?.search) params.set('search', filter.search);
  if (filter?.active !== undefined) params.set('active', String(filter.active));
  const queryString = params.toString();

  return useQuery({
    queryKey: [...queryKeys.adminClients, filter],
    queryFn: () =>
      clientApiClient<Client[]>(`/v1/admin/clients${queryString ? `?${queryString}` : ''}`),
    ...options,
  });
}

export function useAdminClient(
  id: string,
  options?: Omit<UseQueryOptions<Client, ApiClientError>, 'queryKey' | 'queryFn'>
) {
  return useQuery({
    queryKey: queryKeys.adminClient(id),
    queryFn: () => clientApiClient<Client>(`/v1/admin/clients/${id}`),
    enabled: !!id,
    ...options,
  });
}
