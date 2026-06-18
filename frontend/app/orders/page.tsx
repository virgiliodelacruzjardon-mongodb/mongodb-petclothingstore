"use client";

import { useEffect, useState, useCallback } from "react";
import { api, ListResponse } from "@/lib/api";
import { Order, Customer, Product, OrderItem } from "@/lib/types";
import Modal from "@/components/Modal";
import Guard from "@/components/Guard";
import { Pagination } from "../products/page";

const STATUSES = ["pending", "paid", "shipped", "delivered", "cancelled"];

const statusColor: Record<string, string> = {
  pending: "bg-gray-100 text-gray-700",
  paid: "bg-blue-100 text-blue-700",
  shipped: "bg-amber-100 text-amber-700",
  delivered: "bg-green-100 text-green-700",
  cancelled: "bg-red-100 text-red-700",
};

const empty = (): any => ({
  customer: { id: "", name: "", email: "" },
  items: [],
  status: "pending",
  pricing: { discount: 0 },
  shippingAddress: { line1: "", city: "", state: "", zip: "", country: "US" },
});

function OrdersInner() {
  const [items, setItems] = useState<Order[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<any>(empty());
  const [err, setErr] = useState("");
  const limit = 15;

  const load = useCallback(async () => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    if (status) params.set("status", status);
    const r = await api.get<ListResponse<Order>>(`/orders?${params}`);
    setItems(r.data || []);
    setTotal(r.total);
  }, [page, status]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    api.get<ListResponse<Customer>>("/customers?limit=100").then((r) => setCustomers(r.data || []));
    api.get<ListResponse<Product>>("/products?limit=100").then((r) => setProducts(r.data || []));
  }, []);

  function addItem() {
    setEditing({
      ...editing,
      items: [
        ...editing.items,
        { productId: "", name: "", sku: "", size: "", color: "", price: 0, qty: 1 },
      ],
    });
  }
  function setItemProduct(i: number, productId: string) {
    const p = products.find((x) => x.id === productId);
    const items = [...editing.items];
    if (p) {
      const v = p.variants?.[0];
      items[i] = {
        ...items[i],
        productId: p.id,
        name: p.name,
        sku: v?.sku || "",
        size: v?.size || "",
        color: v?.color || "",
        price: v?.price || p.basePrice,
      };
    }
    setEditing({ ...editing, items });
  }
  function setQty(i: number, qty: number) {
    const items = [...editing.items];
    items[i] = { ...items[i], qty };
    setEditing({ ...editing, items });
  }
  function removeItem(i: number) {
    const items = [...editing.items];
    items.splice(i, 1);
    setEditing({ ...editing, items });
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    if (!editing.customer?.id) {
      setErr("Select a customer");
      return;
    }
    if (!editing.items.length) {
      setErr("Add at least one item");
      return;
    }
    try {
      if (editing.id) await api.put(`/orders/${editing.id}`, editing);
      else await api.post("/orders", editing);
      setOpen(false);
      load();
    } catch (e: any) {
      setErr(e.message);
    }
  }
  async function remove(id: string) {
    if (!confirm("Delete this order?")) return;
    await api.del(`/orders/${id}`);
    load();
  }

  const estSubtotal = (editing.items as OrderItem[]).reduce(
    (s, it) => s + (it.price || 0) * (it.qty || 0),
    0
  );

  const pages = Math.ceil(total / limit) || 1;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">
          Orders <span className="text-base font-normal text-gray-400">({total})</span>
        </h1>
        <button
          className="btn-primary"
          onClick={() => {
            setEditing(empty());
            setErr("");
            setOpen(true);
          }}
        >
          + New order
        </button>
      </div>

      <select
        className="input mb-4 max-w-[180px]"
        value={status}
        onChange={(e) => {
          setPage(1);
          setStatus(e.target.value);
        }}
      >
        <option value="">All statuses</option>
        {STATUSES.map((s) => (
          <option key={s}>{s}</option>
        ))}
      </select>

      <div className="card overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-left text-xs uppercase text-gray-500">
            <tr>
              <th className="px-4 py-3">Order #</th>
              <th className="px-4 py-3">Customer</th>
              <th className="px-4 py-3">Items</th>
              <th className="px-4 py-3">Total</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {items.map((o) => (
              <tr key={o.id}>
                <td className="px-4 py-3 font-medium">{o.orderNumber}</td>
                <td className="px-4 py-3 text-gray-600">{o.customer?.name}</td>
                <td className="px-4 py-3">{o.items?.length}</td>
                <td className="px-4 py-3 font-semibold">
                  ${o.pricing?.total?.toFixed(2)}
                </td>
                <td className="px-4 py-3">
                  <span className={`badge ${statusColor[o.status] || ""}`}>
                    {o.status}
                  </span>
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="flex justify-end gap-2">
                    <button
                      className="btn-ghost"
                      onClick={() => {
                        setEditing(JSON.parse(JSON.stringify(o)));
                        setErr("");
                        setOpen(true);
                      }}
                    >
                      Edit
                    </button>
                    <button className="btn-danger" onClick={() => remove(o.id)}>
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pages={pages} onChange={setPage} />

      <Modal
        open={open}
        title={editing.id ? `Edit ${editing.orderNumber || "order"}` : "New order"}
        onClose={() => setOpen(false)}
      >
        <form onSubmit={save} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Customer</label>
              <select
                className="input"
                value={editing.customer?.id || ""}
                onChange={(e) => {
                  const c = customers.find((x) => x.id === e.target.value);
                  setEditing({
                    ...editing,
                    customer: c
                      ? {
                          id: c.id,
                          name: `${c.firstName} ${c.lastName}`,
                          email: c.email,
                        }
                      : { id: "", name: "", email: "" },
                  });
                }}
                required
              >
                <option value="">Select…</option>
                {customers.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.firstName} {c.lastName} — {c.email}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Status</label>
              <select
                className="input"
                value={editing.status}
                onChange={(e) =>
                  setEditing({ ...editing, status: e.target.value })
                }
              >
                {STATUSES.map((s) => (
                  <option key={s}>{s}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <div className="mb-2 flex items-center justify-between">
              <label className="label mb-0">Items</label>
              <button type="button" className="text-sm text-brand-600" onClick={addItem}>
                + Add item
              </button>
            </div>
            <div className="space-y-2">
              {editing.items.map((it: OrderItem, i: number) => (
                <div key={i} className="grid grid-cols-12 gap-2">
                  <select
                    className="input col-span-7"
                    value={it.productId}
                    onChange={(e) => setItemProduct(i, e.target.value)}
                  >
                    <option value="">Select product…</option>
                    {products.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name} (${p.basePrice?.toFixed(2)})
                      </option>
                    ))}
                  </select>
                  <input
                    className="input col-span-2"
                    type="number"
                    min={1}
                    value={it.qty}
                    onChange={(e) => setQty(i, Number(e.target.value))}
                  />
                  <span className="col-span-2 self-center text-sm text-gray-600">
                    ${((it.price || 0) * (it.qty || 0)).toFixed(2)}
                  </span>
                  <button
                    type="button"
                    className="col-span-1 text-red-500"
                    onClick={() => removeItem(i)}
                  >
                    ✕
                  </button>
                </div>
              ))}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Discount ($)</label>
              <input
                className="input"
                type="number"
                step="0.01"
                value={editing.pricing?.discount || 0}
                onChange={(e) =>
                  setEditing({
                    ...editing,
                    pricing: { ...editing.pricing, discount: Number(e.target.value) },
                  })
                }
              />
            </div>
            <div className="self-end text-right text-sm text-gray-600">
              Est. subtotal: <span className="font-semibold">${estSubtotal.toFixed(2)}</span>
              <p className="text-xs text-gray-400">
                Tax, shipping &amp; total are computed by the server.
              </p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Ship to (address)</label>
              <input
                className="input"
                placeholder="Street"
                value={editing.shippingAddress?.line1 || ""}
                onChange={(e) =>
                  setEditing({
                    ...editing,
                    shippingAddress: { ...editing.shippingAddress, line1: e.target.value },
                  })
                }
              />
            </div>
            <div>
              <label className="label">City</label>
              <input
                className="input"
                value={editing.shippingAddress?.city || ""}
                onChange={(e) =>
                  setEditing({
                    ...editing,
                    shippingAddress: { ...editing.shippingAddress, city: e.target.value },
                  })
                }
              />
            </div>
          </div>

          {err && <p className="text-sm text-red-600">{err}</p>}
          <div className="flex justify-end gap-2">
            <button type="button" className="btn-ghost" onClick={() => setOpen(false)}>
              Cancel
            </button>
            <button className="btn-primary">Save</button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

export default function OrdersPage() {
  return (
    <Guard>
      <OrdersInner />
    </Guard>
  );
}
