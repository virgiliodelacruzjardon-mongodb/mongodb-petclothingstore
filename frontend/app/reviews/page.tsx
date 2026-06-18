"use client";

import { useEffect, useState, useCallback } from "react";
import { api, ListResponse } from "@/lib/api";
import { Review } from "@/lib/types";
import { useAuth } from "@/lib/auth";
import { Pagination } from "../products/page";

function Stars({ n }: { n: number }) {
  return (
    <span className="text-amber-500">
      {"★".repeat(n)}
      <span className="text-gray-300">{"★".repeat(5 - n)}</span>
    </span>
  );
}

export default function ReviewsPage() {
  const { user } = useAuth();
  const [items, setItems] = useState<Review[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const limit = 15;

  const load = useCallback(async () => {
    const r = await api.get<ListResponse<Review>>(
      `/reviews?page=${page}&limit=${limit}`
    );
    setItems(r.data || []);
    setTotal(r.total);
  }, [page]);

  useEffect(() => {
    load();
  }, [load]);

  async function remove(id: string) {
    if (!confirm("Delete this review?")) return;
    await api.del(`/reviews/${id}`);
    load();
  }

  const pages = Math.ceil(total / limit) || 1;

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">
        Reviews <span className="text-base font-normal text-gray-400">({total})</span>
      </h1>

      <div className="space-y-3">
        {items.map((r) => (
          <div key={r.id} className="card p-4">
            <div className="flex items-start justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <Stars n={r.rating} />
                  <span className="font-semibold">{r.title}</span>
                  {r.verified && (
                    <span className="badge bg-green-100 text-green-700">
                      verified
                    </span>
                  )}
                </div>
                <p className="mt-1 text-sm text-gray-600">{r.body}</p>
                <p className="mt-2 text-xs text-gray-400">
                  {r.customer?.name} on{" "}
                  <span className="font-medium text-gray-500">
                    {r.product?.name}
                  </span>
                </p>
              </div>
              {user && (
                <button className="btn-danger" onClick={() => remove(r.id)}>
                  Delete
                </button>
              )}
            </div>
          </div>
        ))}
      </div>

      <Pagination page={page} pages={pages} onChange={setPage} />
    </div>
  );
}
