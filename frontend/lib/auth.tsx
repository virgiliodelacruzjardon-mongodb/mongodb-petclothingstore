"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { api } from "./api";
import { Customer } from "./types";

interface AuthState {
  user: Customer | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (
    firstName: string,
    lastName: string,
    email: string,
    password: string
  ) => Promise<void>;
  logout: () => void;
}

const AuthCtx = createContext<AuthState>({} as AuthState);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<Customer | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const t = localStorage.getItem("token");
    if (!t) {
      setLoading(false);
      return;
    }
    api
      .get<Customer>("/auth/me")
      .then(setUser)
      .catch(() => localStorage.removeItem("token"))
      .finally(() => setLoading(false));
  }, []);

  async function login(email: string, password: string) {
    const res = await api.post<{ token: string; customer: Customer }>(
      "/auth/login",
      { email, password }
    );
    localStorage.setItem("token", res.token);
    setUser(res.customer);
  }

  async function register(
    firstName: string,
    lastName: string,
    email: string,
    password: string
  ) {
    const res = await api.post<{ token: string; customer: Customer }>(
      "/auth/register",
      { firstName, lastName, email, password }
    );
    localStorage.setItem("token", res.token);
    setUser(res.customer);
  }

  function logout() {
    localStorage.removeItem("token");
    setUser(null);
  }

  return (
    <AuthCtx.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthCtx.Provider>
  );
}

export const useAuth = () => useContext(AuthCtx);
