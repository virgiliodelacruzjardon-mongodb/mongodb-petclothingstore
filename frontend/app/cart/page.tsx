"use client";

import { useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { useCart } from "@/lib/cart";
import { useAuth } from "@/lib/auth";
import { Order } from "@/lib/types";

export default function CartPage() {
  const { items, setQty, remove, clear, subtotal, count } = useCart();
  const { user } = useAuth();

  const [addr, setAddr] = useState({
    line1: "",
    city: "",
    state: "",
    zip: "",
    country: "US",
  });
  const [placing, setPlacing] = useState(false);
  const [err, setErr] = useState("");
  const [placed, setPlaced] = useState<Order | null>(null);

  // rough preview; the server is the source of truth for totals
  const shipping = subtotal > 75 || subtotal === 0 ? 0 : 6.99;
  const tax = subtotal * 0.08;
  const total = subtotal + tax + shipping;

  async function checkout() {
    if (!user) return;
    setErr("");
    setPlacing(true);
    try {
      const payload = {
        customer: {
          id: user.id,
          name: `${user.firstName} ${user.lastName}`,
          email: user.email,
        },
        items: items.map((it) => ({
          productId: it.productId,
          name: it.name,
          sku: it.sku,
          size: it.size,
          color: it.color,
          price: it.price,
          qty: it.qty,
        })),
        shippingAddress: { ...addr, label: "Shipping", default: true },
        pricing: { discount: 0 },
        status: "paid",
      };
      const order = await api.post<Order>("/orders", payload);
      clear();
      setPlaced(order);
    } catch (e: any) {
      setErr(e.message || "Checkout failed");
    } finally {
      setPlacing(false);
    }
  }

  if (placed) {
    return (
      <div className="mx-auto max-w-lg text-center">
        <div className="card p-10">
          <div className="text-5xl">🎉</div>
          <h1 className="mt-4 text-2xl font-bold">Order placed!</h1>
          <p className="mt-2 text-gray-600">
            Order <span className="font-mono font-semibold">{placed.orderNumber}</span> —
            total <span className="font-semibold">${placed.pricing?.total?.toFixed(2)}</span>.
          </p>
          <p className="mt-1 text-sm text-gray-500">
            Stock was decremented atomically via a MongoDB transaction.
          </p>
          <Link href="/shop" className="btn-primary mt-6 inline-flex">
            Continue shopping
          </Link>
        </div>
      </div>
    );
  }

  if (count === 0) {
    return (
      <div className="mx-auto max-w-lg text-center">
        <div className="card p-10">
          <div className="text-4xl">🛒</div>
          <h1 className="mt-4 text-xl font-bold">Your cart is empty</h1>
          <Link href="/shop" className="btn-primary mt-6 inline-flex">
            Browse the shop
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      {/* items */}
      <div className="lg:col-span-2">
        <h1 className="mb-4 text-2xl font-bold">Your cart ({count})</h1>
        <div className="card divide-y divide-gray-100">
          {items.map((it) => (
            <div key={it.sku} className="flex items-center gap-4 p-4">
              {it.image ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={it.image}
                  alt={it.name}
                  className="h-16 w-16 rounded-lg object-cover"
                />
              ) : (
                <div className="h-16 w-16 rounded-lg bg-gray-100" />
              )}
              <div className="flex-1">
                <p className="font-medium">{it.name}</p>
                <p className="text-sm text-gray-500">
                  {it.size} / {it.color} · {it.sku}
                </p>
                <p className="text-sm font-semibold">${it.price.toFixed(2)}</p>
              </div>
              <input
                type="number"
                min={1}
                max={it.maxStock}
                value={it.qty}
                onChange={(e) => setQty(it.sku, Number(e.target.value))}
                className="input w-20"
              />
              <div className="w-20 text-right font-semibold">
                ${(it.price * it.qty).toFixed(2)}
              </div>
              <button
                className="text-red-500 hover:text-red-700"
                onClick={() => remove(it.sku)}
              >
                ✕
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* summary + checkout */}
      <div>
        <div className="card sticky top-6 p-5">
          <h2 className="mb-4 text-lg font-semibold">Order summary</h2>
          <dl className="space-y-1 text-sm">
            <Row label="Subtotal" value={subtotal} />
            <Row label="Tax (8%)" value={tax} />
            <Row label="Shipping" value={shipping} note={shipping === 0 ? "FREE" : undefined} />
            <div className="my-2 border-t border-gray-100" />
            <div className="flex justify-between text-base font-bold">
              <span>Total</span>
              <span>${total.toFixed(2)}</span>
            </div>
          </dl>
          <p className="mt-1 text-xs text-gray-400">
            Final total is computed by the server. Free shipping over $75.
          </p>

          {user ? (
            <div className="mt-5 space-y-3">
              <h3 className="text-sm font-semibold">Ship to</h3>
              <input
                className="input"
                placeholder="Street address"
                value={addr.line1}
                onChange={(e) => setAddr({ ...addr, line1: e.target.value })}
              />
              <div className="grid grid-cols-2 gap-2">
                <input
                  className="input"
                  placeholder="City"
                  value={addr.city}
                  onChange={(e) => setAddr({ ...addr, city: e.target.value })}
                />
                <input
                  className="input"
                  placeholder="State"
                  value={addr.state}
                  onChange={(e) => setAddr({ ...addr, state: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <input
                  className="input"
                  placeholder="ZIP"
                  value={addr.zip}
                  onChange={(e) => setAddr({ ...addr, zip: e.target.value })}
                />
                <input
                  className="input"
                  placeholder="Country"
                  value={addr.country}
                  onChange={(e) => setAddr({ ...addr, country: e.target.value })}
                />
              </div>

              {err && <p className="text-sm text-red-600">{err}</p>}

              <button
                className="btn-primary w-full"
                disabled={placing}
                onClick={checkout}
              >
                {placing ? "Placing order…" : "Place order"}
              </button>
            </div>
          ) : (
            <div className="mt-5 rounded-lg bg-brand-50 p-4 text-sm text-gray-700">
              Please{" "}
              <Link href="/login" className="font-semibold text-brand-700 underline">
                log in
              </Link>{" "}
              to check out. (admin@petstore.dev / admin123)
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function Row({
  label,
  value,
  note,
}: {
  label: string;
  value: number;
  note?: string;
}) {
  return (
    <div className="flex justify-between">
      <span className="text-gray-600">{label}</span>
      <span>{note || `$${value.toFixed(2)}`}</span>
    </div>
  );
}
