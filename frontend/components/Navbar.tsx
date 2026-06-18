"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { useCart } from "@/lib/cart";

const shopLinks = [{ href: "/shop", label: "Shop" }];

const adminLinks = [
  { href: "/", label: "Dashboard" },
  { href: "/products", label: "Products" },
  { href: "/categories", label: "Categories" },
  { href: "/customers", label: "Customers" },
  { href: "/orders", label: "Orders" },
  { href: "/reviews", label: "Reviews" },
];

export default function Navbar() {
  const path = usePathname();
  const { user, logout } = useAuth();
  const { count } = useCart();

  const links = [...shopLinks, ...adminLinks];

  return (
    <header className="border-b border-gray-200 bg-white">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
        <div className="flex items-center gap-6">
          <Link href="/shop" className="text-lg font-bold text-brand-600">
            🐾 PawCouture
          </Link>
          <nav className="hidden gap-1 md:flex">
            {links.map((l) => {
              const active = path === l.href;
              return (
                <Link
                  key={l.href}
                  href={l.href}
                  className={`rounded-lg px-3 py-1.5 text-sm font-medium ${
                    active
                      ? "bg-brand-50 text-brand-700"
                      : "text-gray-600 hover:bg-gray-100"
                  }`}
                >
                  {l.label}
                </Link>
              );
            })}
          </nav>
        </div>

        <div className="flex items-center gap-3">
          <Link
            href="/cart"
            className="relative rounded-lg px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100"
          >
            🛒 Cart
            {count > 0 && (
              <span className="absolute -right-1 -top-1 flex h-5 min-w-[20px] items-center justify-center rounded-full bg-brand-600 px-1 text-xs font-bold text-white">
                {count}
              </span>
            )}
          </Link>
          {user ? (
            <>
              <span className="hidden text-sm text-gray-600 sm:block">
                {user.firstName}
                <span className="ml-2 badge bg-brand-100 text-brand-700">
                  {user.role}
                </span>
              </span>
              <button className="btn-ghost" onClick={logout}>
                Logout
              </button>
            </>
          ) : (
            <Link href="/login" className="btn-primary">
              Login
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
