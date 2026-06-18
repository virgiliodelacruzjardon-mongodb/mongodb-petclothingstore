"use client";

import React, { createContext, useContext, useEffect, useState } from "react";

// A cart line is variant-level: the SKU uniquely identifies it.
export interface CartItem {
  productId: string;
  name: string;
  sku: string;
  size: string;
  color: string;
  price: number;
  image: string;
  qty: number;
  maxStock: number;
}

interface CartState {
  items: CartItem[];
  add: (it: CartItem) => void;
  remove: (sku: string) => void;
  setQty: (sku: string, qty: number) => void;
  clear: () => void;
  count: number;
  subtotal: number;
}

const CartCtx = createContext<CartState>({} as CartState);
const KEY = "cart";

export function CartProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<CartItem[]>([]);

  useEffect(() => {
    try {
      const raw = localStorage.getItem(KEY);
      if (raw) setItems(JSON.parse(raw));
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    localStorage.setItem(KEY, JSON.stringify(items));
  }, [items]);

  function add(it: CartItem) {
    setItems((cur) => {
      const i = cur.findIndex((x) => x.sku === it.sku);
      if (i >= 0) {
        const next = [...cur];
        next[i] = {
          ...next[i],
          qty: Math.min(next[i].qty + it.qty, it.maxStock || 99),
        };
        return next;
      }
      return [...cur, it];
    });
  }
  function remove(sku: string) {
    setItems((cur) => cur.filter((x) => x.sku !== sku));
  }
  function setQty(sku: string, qty: number) {
    setItems((cur) =>
      cur.map((x) =>
        x.sku === sku
          ? { ...x, qty: Math.max(1, Math.min(qty, x.maxStock || 99)) }
          : x
      )
    );
  }
  function clear() {
    setItems([]);
  }

  const count = items.reduce((s, x) => s + x.qty, 0);
  const subtotal = items.reduce((s, x) => s + x.price * x.qty, 0);

  return (
    <CartCtx.Provider value={{ items, add, remove, setQty, clear, count, subtotal }}>
      {children}
    </CartCtx.Provider>
  );
}

export const useCart = () => useContext(CartCtx);
