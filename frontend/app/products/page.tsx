"use client";

import { useEffect, useState, useCallback } from "react";
import { api, ListResponse } from "@/lib/api";
import { Product, Category, Variant } from "@/lib/types";
import { useAuth } from "@/lib/auth";
import Modal from "@/components/Modal";

const PETS = ["dog", "cat", "small-pet"];
const SIZES = ["XS", "S", "M", "L", "XL"];

const emptyProduct = (): Partial<Product> => ({
  name: "",
  brand: "",
  petType: "dog",
  description: "",
  basePrice: 0,
  currency: "USD",
  active: true,
  tags: [],
  images: [],
  variants: [{ sku: "", size: "M", color: "Black", price: 0, stock: 0 }],
  category: { id: "", name: "", slug: "" },
});

export default function ProductsPage() {
  const { user } = useAuth();
  const [items, setItems] = useState<Product[]>([]);
  const [cats, setCats] = useState<Category[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState("");
  const [petType, setPetType] = useState("");
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Partial<Product>>(emptyProduct());
  const [err, setErr] = useState("");
  const limit = 12;

  const load = useCallback(async () => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    if (q) params.set("q", q);
    if (petType) params.set("petType", petType);
    const r = await api.get<ListResponse<Product>>(`/products?${params}`);
    setItems(r.data || []);
    setTotal(r.total);
  }, [page, q, petType]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    api
      .get<ListResponse<Category>>("/categories?limit=100")
      .then((r) => setCats(r.data || []));
  }, []);

  function openNew() {
    setEditing(emptyProduct());
    setErr("");
    setOpen(true);
  }
  function openEdit(p: Product) {
    setEditing(JSON.parse(JSON.stringify(p)));
    setErr("");
    setOpen(true);
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      const payload = { ...editing, basePrice: Number(editing.basePrice) };
      if (editing.id) {
        await api.put(`/products/${editing.id}`, payload);
      } else {
        await api.post("/products", payload);
      }
      setOpen(false);
      load();
    } catch (e: any) {
      setErr(e.message);
    }
  }

  async function remove(id: string) {
    if (!confirm("Delete this product?")) return;
    await api.del(`/products/${id}`);
    load();
  }

  function setVariant(i: number, key: keyof Variant, value: string | number) {
    const variants = [...(editing.variants || [])];
    (variants[i] as any)[key] = value;
    setEditing({ ...editing, variants });
  }
  function addVariant() {
    setEditing({
      ...editing,
      variants: [
        ...(editing.variants || []),
        { sku: "", size: "M", color: "Black", price: 0, stock: 0 },
      ],
    });
  }
  function removeVariant(i: number) {
    const variants = [...(editing.variants || [])];
    variants.splice(i, 1);
    setEditing({ ...editing, variants });
  }

  const pages = Math.ceil(total / limit) || 1;

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">Products <span className="text-base font-normal text-gray-400">({total})</span></h1>
        {user && (
          <button className="btn-primary" onClick={openNew}>
            + New product
          </button>
        )}
      </div>

      <div className="mb-4 flex flex-wrap gap-3">
        <input
          className="input max-w-xs"
          placeholder="Search name or tags…"
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
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {items.map((p) => (
          <div key={p.id} className="card overflow-hidden">
            {p.images?.[0] && (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={p.images[0]}
                alt={p.name}
                className="h-40 w-full object-cover"
              />
            )}
            <div className="p-4">
              <div className="flex items-start justify-between gap-2">
                <h3 className="font-semibold leading-tight">{p.name}</h3>
                <span className="badge bg-brand-100 text-brand-700">
                  {p.petType}
                </span>
              </div>
              <p className="mt-1 text-sm text-gray-500">{p.category?.name}</p>
              <div className="mt-2 flex items-center justify-between">
                <span className="text-lg font-bold">
                  ${p.basePrice?.toFixed(2)}
                </span>
                <span className="text-sm text-amber-600">
                  ★ {p.ratingSummary?.avg ?? 0} ({p.ratingSummary?.count ?? 0})
                </span>
              </div>
              <p className="mt-1 text-xs text-gray-500">
                Stock: {p.totalStock} · {p.variants?.length ?? 0} variants
              </p>
              {user && (
                <div className="mt-3 flex gap-2">
                  <button className="btn-ghost flex-1" onClick={() => openEdit(p)}>
                    Edit
                  </button>
                  <button
                    className="btn-danger"
                    onClick={() => remove(p.id)}
                  >
                    Delete
                  </button>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      <Pagination page={page} pages={pages} onChange={setPage} />

      <Modal
        open={open}
        title={editing.id ? "Edit product" : "New product"}
        onClose={() => setOpen(false)}
      >
        <form onSubmit={save} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Name</label>
              <input
                className="input"
                value={editing.name || ""}
                onChange={(e) =>
                  setEditing({ ...editing, name: e.target.value })
                }
                required
              />
            </div>
            <div>
              <label className="label">Brand</label>
              <input
                className="input"
                value={editing.brand || ""}
                onChange={(e) =>
                  setEditing({ ...editing, brand: e.target.value })
                }
              />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="label">Pet type</label>
              <select
                className="input"
                value={editing.petType}
                onChange={(e) =>
                  setEditing({ ...editing, petType: e.target.value })
                }
              >
                {PETS.map((p) => (
                  <option key={p}>{p}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Base price</label>
              <input
                className="input"
                type="number"
                step="0.01"
                value={editing.basePrice}
                onChange={(e) =>
                  setEditing({ ...editing, basePrice: Number(e.target.value) })
                }
              />
            </div>
            <div>
              <label className="label">Category</label>
              <select
                className="input"
                value={editing.category?.id || ""}
                onChange={(e) => {
                  const c = cats.find((x) => x.id === e.target.value);
                  setEditing({
                    ...editing,
                    category: c
                      ? { id: c.id, name: c.name, slug: c.slug }
                      : { id: "", name: "", slug: "" },
                  });
                }}
                required
              >
                <option value="">Select…</option>
                {cats.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="label">Description</label>
            <textarea
              className="input"
              rows={2}
              value={editing.description || ""}
              onChange={(e) =>
                setEditing({ ...editing, description: e.target.value })
              }
            />
          </div>

          <div>
            <label className="label">Tags (comma separated)</label>
            <input
              className="input"
              value={(editing.tags || []).join(", ")}
              onChange={(e) =>
                setEditing({
                  ...editing,
                  tags: e.target.value
                    .split(",")
                    .map((t) => t.trim())
                    .filter(Boolean),
                })
              }
            />
          </div>

          <div>
            <label className="label">Image URL</label>
            <input
              className="input"
              value={editing.images?.[0] || ""}
              onChange={(e) =>
                setEditing({ ...editing, images: e.target.value ? [e.target.value] : [] })
              }
            />
          </div>

          <div>
            <div className="mb-2 flex items-center justify-between">
              <label className="label mb-0">Variants (embedded)</label>
              <button type="button" className="text-sm text-brand-600" onClick={addVariant}>
                + Add variant
              </button>
            </div>
            <div className="space-y-2">
              {(editing.variants || []).map((v, i) => (
                <div key={i} className="grid grid-cols-12 gap-2">
                  <input
                    className="input col-span-3"
                    placeholder="SKU"
                    value={v.sku}
                    onChange={(e) => setVariant(i, "sku", e.target.value)}
                  />
                  <select
                    className="input col-span-2"
                    value={v.size}
                    onChange={(e) => setVariant(i, "size", e.target.value)}
                  >
                    {SIZES.map((s) => (
                      <option key={s}>{s}</option>
                    ))}
                  </select>
                  <input
                    className="input col-span-2"
                    placeholder="Color"
                    value={v.color}
                    onChange={(e) => setVariant(i, "color", e.target.value)}
                  />
                  <input
                    className="input col-span-2"
                    type="number"
                    step="0.01"
                    placeholder="Price"
                    value={v.price}
                    onChange={(e) => setVariant(i, "price", Number(e.target.value))}
                  />
                  <input
                    className="input col-span-2"
                    type="number"
                    placeholder="Stock"
                    value={v.stock}
                    onChange={(e) => setVariant(i, "stock", Number(e.target.value))}
                  />
                  <button
                    type="button"
                    className="col-span-1 text-red-500"
                    onClick={() => removeVariant(i)}
                  >
                    ✕
                  </button>
                </div>
              ))}
            </div>
          </div>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={!!editing.active}
              onChange={(e) =>
                setEditing({ ...editing, active: e.target.checked })
              }
            />
            Active (visible in store)
          </label>

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

export function Pagination({
  page,
  pages,
  onChange,
}: {
  page: number;
  pages: number;
  onChange: (p: number) => void;
}) {
  if (pages <= 1) return null;
  return (
    <div className="mt-6 flex items-center justify-center gap-2">
      <button
        className="btn-ghost"
        disabled={page <= 1}
        onClick={() => onChange(page - 1)}
      >
        Prev
      </button>
      <span className="text-sm text-gray-600">
        Page {page} of {pages}
      </span>
      <button
        className="btn-ghost"
        disabled={page >= pages}
        onClick={() => onChange(page + 1)}
      >
        Next
      </button>
    </div>
  );
}
