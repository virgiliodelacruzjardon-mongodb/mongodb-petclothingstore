"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, ListResponse } from "@/lib/api";
import { useAuth } from "@/lib/auth";

interface Stat {
  label: string;
  href: string;
  total: number | string;
  icon: string;
}

export default function Dashboard() {
  const { user } = useAuth();
  const [stats, setStats] = useState<Stat[]>([
    { label: "Products", href: "/products", total: "…", icon: "🧥" },
    { label: "Categories", href: "/categories", total: "…", icon: "🏷️" },
    { label: "Customers", href: "/customers", total: "…", icon: "🧑" },
    { label: "Orders", href: "/orders", total: "…", icon: "📦" },
    { label: "Reviews", href: "/reviews", total: "…", icon: "⭐" },
  ]);

  useEffect(() => {
    async function load() {
      const endpoints = [
        ["/products?limit=1", "Products"],
        ["/categories?limit=1", "Categories"],
        ["/reviews?limit=1", "Reviews"],
      ];
      const authed = [
        ["/customers?limit=1", "Customers"],
        ["/orders?limit=1", "Orders"],
      ];
      const all = user ? [...endpoints, ...authed] : endpoints;
      const next = [...stats];
      for (const [path, label] of all) {
        try {
          const r = await api.get<ListResponse<unknown>>(path);
          const idx = next.findIndex((s) => s.label === label);
          if (idx >= 0) next[idx] = { ...next[idx], total: r.total };
        } catch {
          /* ignore (likely needs auth) */
        }
      }
      setStats(next);
    }
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user]);

  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold">Store Dashboard</h1>
      <p className="mb-8 text-gray-500">
        Next.js + Go + MongoDB 8 — pet clothing CRUD demo.
      </p>

      <div className="grid grid-cols-2 gap-4 md:grid-cols-5">
        {stats.map((s) => (
          <Link key={s.label} href={s.href} className="card p-5 hover:shadow-md">
            <div className="text-3xl">{s.icon}</div>
            <div className="mt-3 text-2xl font-bold">{s.total}</div>
            <div className="text-sm text-gray-500">{s.label}</div>
          </Link>
        ))}
      </div>

      {!user && (
        <div className="card mt-8 p-6">
          <p className="text-gray-700">
            Log in (admin@petstore.dev / admin123) to manage customers and
            orders and to create/edit/delete records.
          </p>
        </div>
      )}
    </div>
  );
}
