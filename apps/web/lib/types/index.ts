// Types matching the Go backend models

export type OrderStatus =
  | 'draft'
  | 'pending'
  | 'approved'
  | 'rejected'
  | 'shipped'
  | 'delivered'
  | 'cancelled';

export type VATType = 'general' | 'reduced' | 'super_reduced' | 'exempt';

// Product types
export interface ProductImage {
  url: string;
  is_primary: boolean;
  display_order: number;
}

export interface Product {
  id: string;
  sku: string;
  name: string;
  description?: string;
  category?: string;
  price: number;
  tax_rate: number;
  stock_quantity: number;
  min_order_quantity: number;
  images?: ProductImage[];
}

// Cart types
export interface CartItemResponse {
  product_id: string;
  product_name: string;
  product_sku: string;
  quantity: number;
  unit_price: number;
  line_total: number;
  stock_available: number;
  min_order_quantity: number;
  in_stock: boolean;
}

export interface CartSummary {
  subtotal: number;
  tax_rate: number;
  tax_amount: number;
  total: number;
  item_count: number;
  total_units: number;
}

export interface CartResponse {
  id: string;
  status: OrderStatus;
  notes?: string;
  created_at: string;
  updated_at: string;
  items: CartItemResponse[];
  summary: CartSummary;
}

// Order types
export interface OrderItem {
  product_id: string;
  quantity: number;
}

export interface Order {
  id: string;
  status: OrderStatus;
  notes?: string;
  holded_invoice_id?: string;
  approved_at?: string;
  rejected_at?: string;
  rejection_reason?: string;
  created_at: string;
  items: OrderItem[];
}

// Client types
export interface Address {
  street?: string;
  city?: string;
  state?: string;
  postal_code?: string;
  country?: string;
}

export interface Client {
  id: string;
  holded_id: string;
  email: string;
  company_name: string;
  contact_name?: string;
  phone?: string;
  vat_type?: VATType;
  vat_number?: string;
  billing_address?: Address;
  shipping_address?: Address;
  is_active: boolean;
  is_linked: boolean;
  created_at: string;
}

// API Error types
export interface APIError {
  error: string;
  code?: string;
  message?: string;
}
