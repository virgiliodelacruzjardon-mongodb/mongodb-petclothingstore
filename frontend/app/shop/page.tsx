"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { api, ListResponse } from "@/lib/api";
import { Product, Variant } from "@/lib/types";
import { useCart } from "@/lib/cart";
import { Pagination } from "../products/page";

const PETS = ["dog", "cat", "small-pet"];

export default function ShopPage() {
  const [items, setItems] = useState<Product[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState("");
  const [petType, setPetType] = useState("");
  const limit = 12;

  const load = useCallback(async () => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    if (q) params.set("q", q);
    if (petType) params.set("petType", petType);
    const r = await api.get<ListResponse<Product>>(`/products?${params}`);
    setItems((r.data || []).filter((p) => p.active));
    setTotal(r.total);
  }, [page, q, petType]);

  useEffect(() => {
    load();
  }, [load]);

  const pages = Math.ceil(total / limit) || 1;

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Shop pet clothing 🐾</h1>
        <p className="text-gray-500">
          Pick a size &amp; color, add to cart, and check out.
        </p>
      </div>

      <div className="mb-6 flex flex-wrap gap-3">
        <input
          className="input max-w-xs"
          placeholder="Search…"
          value={q}
          onChange={(e) => {
            setPage(1);
            setQ(e.target.value);
          }}
        />
        <select
          className="input max-w-[160px]"
          value={petType}
          onChange={(e) => {
            setPage(1);
            setPetType(e.target.value);
          }}
        >
          <option value="">All pets</option>
          {PETS.map((p) => (
            <option key={p}>{p}</option>
          ))}
        </select>
        <Link href="/cart" className="btn-ghost ml-auto">
          Go to cart →
        </Link>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {items.map((p) => (
          <ShopCard key={p.id} product={p} />
        ))}
      </div>

      {items.length === 0 && (
        <p className="mt-10 text-center text-gray-500">No products found.</p>
      )}

      <Pagination page={page} pages={pages} onChange={setPage} />
    </div>
  );
}

function ShopCard({ product }: { product: Product }) {
  const { add } = useCart();
  const inStock = (product.variants || []).filter((v) => v.stock > 0);
  const [sku, setSku] = useState(inStock[0]?.sku || "");
  const [qty, setQty] = useState(1);
  const [added, setAdded] = useState(false);

  const variant: Variant | undefined =
    product.variants?.find((v) => v.sku === sku) || inStock[0];

  function handleAdd() {
    if (!variant) return;
    add({
      productId: product.id,
      name: product.name,
      sku: variant.sku,
      size: variant.size,
      color: variant.color,
      price: variant.price,
      image: product.images?.[0] || "",
      qty,
      maxStock: variant.stock,
    });
    setAdded(true);
    setTimeout(() => setAdded(false), 1200);
  }

  return (
    <div className="card flex flex-col overflow-hidden">
      {product.images?.[0] && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={product.images[0]}
          alt={product.name}
          className="h-44 w-full object-cover"
        />
      )}
      <div className="flex flex-1 flex-col p-4">
        <span className="badge mb-1 w-fit bg-brand-100 text-brand-700">
          {product.petType}
        </span>
        <h3 className="font-semibold leading-tight">{product.name}</h3>
        <p className="text-sm text-gray-500">{product.brand}</p>
        <div className="mt-1 flex items-center justify-between">
          <span className="text-lg font-bold">
            ${(variant?.price ?? product.basePrice)?.toFixed(2)}
          </span>
          <span className="text-sm text-amber-600">
            ★ {product.ratingSummary?.avg ?? 0} ({product.ratingSummary?.count ?? 0})
          </span>
        </div>

        <div className="mt-3 space-y-2">
          {inStock.length > 0 ? (
            <>
              <select
                className="input"
                value={sku}
                onChange={(e) => setSku(e.target.value)}
              >
                {inStock.map((v) => (
                  <option key={v.sku} value={v.sku}>
                    {v.size} / {v.color} — ${v.price.toFixed(2)} ({v.stock} left)
                  </option>
                ))}
              </select>
              <div className="flex gap-2">
                <input
                  type="number"
                  min={1}
                  max={variant?.stock || 99}
                  value={qty}
                  onChange={(e) => setQty(Math.max(1, Number(e.target.value)))}
                  className="input w-20"
                />
                <button className="btn-primary flex-1" onClick={handleAdd}>
                  {added ? "Added ✓" : "Add to cart"}
                </button>
              </div>
            </>
          ) : (
            <button className="btn-ghost w-full" disabled>
              Out of stock
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
