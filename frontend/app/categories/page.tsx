"use client";

import { useEffect, useState, useCallback } from "react";
import { api, ListResponse } from "@/lib/api";
import { Category } from "@/lib/types";
import { useAuth } from "@/lib/auth";
import Modal from "@/components/Modal";
import { Pagination } from "../products/page";

const empty = (): Partial<Category> => ({
  name: "",
  slug: "",
  description: "",
  icon: "paw",
});

export default function CategoriesPage() {
  const { user } = useAuth();
  const [items, setItems] = useState<Category[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Partial<Category>>(empty());
  const [err, setErr] = useState("");
  const limit = 20;

  const load = useCallback(async () => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    if (q) params.set("q", q);
    const r = await api.get<ListResponse<Category>>(`/categories?${params}`);
    setItems(r.data || []);
    setTotal(r.total);
  }, [page, q]);

  useEffect(() => {
    load();
  }, [load]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      if (editing.id) await api.put(`/categories/${editing.id}`, editing);
      else await api.post("/categories", editing);
      setOpen(false);
      load();
    } catch (e: any) {
      setErr(e.message);
    }
  }
  async function remove(id: string) {
    if (!confirm("Delete this category?")) return;
    await api.del(`/categories/${id}`);
    load();
  }

  const pages = Math.ceil(total / limit) || 1;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">
          Categories <span className="text-base font-normal text-gray-400">({total})</span>
        </h1>
        {user && (
          <button
            className="btn-primary"
            onClick={() => {
              setEditing(empty());
              setErr("");
              setOpen(true);
            }}
          >
            + New category
          </button>
        )}
      </div>

      <input
        className="input mb-4 max-w-xs"
        placeholder="Search…"
        value={q}
        onChange={(e) => {
          setPage(1);
          setQ(e.target.value);
        }}
      />

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-left text-xs uppercase text-gray-500">
            <tr>
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Slug</th>
              <th className="px-4 py-3">Products</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {items.map((c) => (
              <tr key={c.id}>
                <td className="px-4 py-3 font-medium">{c.name}</td>
                <td className="px-4 py-3 text-gray-500">{c.slug}</td>
                <td className="px-4 py-3">
                  <span className="badge bg-gray-100 text-gray-700">
                    {c.productCount}
                  </span>
                </td>
                <td className="px-4 py-3 text-right">
                  {user && (
                    <div className="flex justify-end gap-2">
                      <button
                        className="btn-ghost"
                        onClick={() => {
                          setEditing({ ...c });
                          setErr("");
                          setOpen(true);
                        }}
                      >
                        Edit
                      </button>
                      <button className="btn-danger" onClick={() => remove(c.id)}>
                        Delete
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pages={pages} onChange={setPage} />

      <Modal
        open={open}
        title={editing.id ? "Edit category" : "New category"}
        onClose={() => setOpen(false)}
      >
        <form onSubmit={save} className="space-y-4">
          <div>
            <label className="label">Name</label>
            <input
              className="input"
              value={editing.name || ""}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="label">Slug (optional)</label>
            <input
              className="input"
              value={editing.slug || ""}
              onChange={(e) => setEditing({ ...editing, slug: e.target.value })}
              placeholder="auto-generated from name if empty"
            />
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
