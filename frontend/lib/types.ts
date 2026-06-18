export interface Category {
  id: string;
  name: string;
  slug: string;
  description: string;
  icon: string;
  productCount: number;
}

export interface Variant {
  sku: string;
  size: string;
  color: string;
  price: number;
  stock: number;
}

export interface RatingSummary {
  avg: number;
  count: number;
  distribution: Record<string, number>;
}

export interface CategoryRef {
  id: string;
  name: string;
  slug: string;
}

export interface Product {
  id: string;
  name: string;
  slug: string;
  description: string;
  brand: string;
  petType: string;
  category: CategoryRef;
  basePrice: number;
  currency: string;
  variants: Variant[];
  images: string[];
  tags: string[];
  totalStock: number;
  ratingSummary: RatingSummary;
  active: boolean;
}

export interface Address {
  label: string;
  line1: string;
  city: string;
  state: string;
  zip: string;
  country: string;
  default: boolean;
}

export interface Customer {
  id: string;
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  role: string;
  addresses: Address[];
  stats: { orderCount: number; totalSpent: number };
}

export interface OrderItem {
  productId: string;
  name: string;
  sku: string;
  size: string;
  color: string;
  price: number;
  qty: number;
  lineTotal: number;
}

export interface Order {
  id: string;
  orderNumber: string;
  customer: { id: string; name: string; email: string };
  items: OrderItem[];
  pricing: {
    subtotal: number;
    tax: number;
    shipping: number;
    discount: number;
    total: number;
  };
  status: string;
  shippingAddress: Address;
  placedAt: string;
}

export interface Review {
  id: string;
  product: { id: string; name: string; slug: string; image: string };
  customer: { id: string; name: string; email: string };
  rating: number;
  title: string;
  body: string;
  verified: boolean;
  createdAt: string;
}
