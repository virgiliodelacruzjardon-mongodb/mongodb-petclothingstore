"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth";

// Wraps pages that require an authenticated session.
export default function Guard({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();

  if (loading) {
    return <p className="text-gray-500">Loading…</p>;
  }
  if (!user) {
    return (
      <div className="card mx-auto max-w-md p-8 text-center">
        <p className="mb-4 text-gray-700">
          You must be logged in to manage the store.
        </p>
        <Link href="/login" className="btn-primary">
          Go to login
        </Link>
      </div>
    );
  }
  return <>{children}</>;
}
