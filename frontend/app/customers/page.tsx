"use client";

import { useEffect, useState, useCallback } from "react";
import { api, ListResponse } from "@/lib/api";
import { Customer } from "@/lib/types";
import Modal from "@/components/Modal";
import Guard from "@/components/Guard";
import { Pagination } from "../products/page";

const empty = (): any => ({
  firstName: "",
  lastName: "",
  email: "",
  phone: "",
  role: "customer",
  password: "",
  addresses: [],
});

function CustomersInner() {
  const [items, setItems] = useState<Customer[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<any>(empty());
  const [err, setErr] = useState("");
  const limit = 20;

  const load = useCallback(async () => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    if (q) params.set("q", q);
    const r = await api.get<ListResponse<Customer>>(`/customers?${params}`);
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
      if (editing.id) await api.put(`/customers/${editing.id}`, editing);
      else await api.post("/customers", editing);
      setOpen(false);
      load();
    } catch (e: any) {
      setErr(e.message);
    }
  }
  async function remove(id: string) {
    if (!confirm("Delete this customer?")) return;
    await api.del(`/customers/${id}`);
    load();
  }

  const pages = Math.ceil(total / limit) || 1;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">
          Customers <span className="text-base font-normal text-gray-400">({total})</span>
        </h1>
        <button
          className="btn-primary"
          onClick={() => {
            setEditing(empty());
            setErr("");
            setOpen(true);
          }}
        >
          + New customer
        </button>
      </div>

      <input
        className="input mb-4 max-w-xs"
        placeholder="Search name or email…"
        value={q}
        onChange={(e) => {
          setPage(1);
          setQ(e.target.value);
        }}
      />

      <div className="card overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-left text-xs uppercase text-gray-500">
            <tr>
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Email</th>
              <th className="px-4 py-3">Role</th>
              <th className="px-4 py-3">Orders</th>
              <th className="px-4 py-3">Spent</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {items.map((c) => (
              <tr key={c.id}>
                <td className="px-4 py-3 font-medium">
                  {c.firstName} {c.lastName}
                </td>
                <td className="px-4 py-3 text-gray-500">{c.email}</td>
                <td className="px-4 py-3">
                  <span className="badge bg-brand-100 text-brand-700">
                    {c.role}
                  </span>
                </td>
                <td className="px-4 py-3">{c.stats?.orderCount ?? 0}</td>
                <td className="px-4 py-3">
                  ${(c.stats?.totalSpent ?? 0).toFixed(2)}
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="flex justify-end gap-2">
                    <button
                      className="btn-ghost"
                      onClick={() => {
                        setEditing({ ...c, password: "" });
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
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pages={pages} onChange={setPage} />

      <Modal
        open={open}
        title={editing.id ? "Edit customer" : "New customer"}
        onClose={() => setOpen(false)}
      >
        <form onSubmit={save} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">First name</label>
              <input
                className="input"
                value={editing.firstName || ""}
                onChange={(e) =>
                  setEditing({ ...editing, firstName: e.target.value })
                }
                required
              />
            </div>
            <div>
              <label className="label">Last name</label>
              <input
                className="input"
                value={editing.lastName || ""}
                onChange={(e) =>
                  setEditing({ ...editing, lastName: e.target.value })
                }
                required
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Email</label>
              <input
                className="input"
                type="email"
                value={editing.email || ""}
                onChange={(e) =>
                  setEditing({ ...editing, email: e.target.value })
                }
                required
              />
            </div>
            <div>
              <label className="label">Phone</label>
              <input
                className="input"
                value={editing.phone || ""}
                onChange={(e) =>
                  setEditing({ ...editing, phone: e.target.value })
                }
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Role</label>
              <select
                className="input"
                value={editing.role}
                onChange={(e) =>
                  setEditing({ ...editing, role: e.target.value })
                }
              >
                <option value="customer">customer</option>
                <option value="admin">admin</option>
              </select>
            </div>
            {!editing.id && (
              <div>
                <label className="label">Password</label>
                <input
                  className="input"
                  type="text"
                  value={editing.password || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, password: e.target.value })
                  }
                  placeholder="defaults to changeme123"
                />
              </div>
            )}
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

export default function CustomersPage() {
  return (
    <Guard>
      <CustomersInner />
    </Guard>
  );
}
