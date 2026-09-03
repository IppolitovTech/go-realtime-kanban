import { createContext, use, useCallback, useState, type ReactNode } from "react";
import { api } from "../api/client";
import type { User } from "../api/types";
import { clearSession, getStoredUser, getToken, setSession } from "./token";

interface AuthContextValue {
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  // Hydrated synchronously from localStorage on first render — see
  // auth/token.ts — so a page reload doesn't flash the login screen before
  // an effect runs.
  const [user, setUser] = useState<User | null>(() => (getToken() ? getStoredUser() : null));

  const login = useCallback(async (email: string, password: string) => {
    const res = await api.login(email, password);
    setSession(res.token, res.user);
    setUser(res.user);
  }, []);

  const register = useCallback(async (email: string, password: string, name: string) => {
    const res = await api.register(email, password, name);
    setSession(res.token, res.user);
    setUser(res.user);
  }, []);

  const logout = useCallback(() => {
    clearSession();
    setUser(null);
  }, []);

  return <AuthContext value={{ user, login, register, logout }}>{children}</AuthContext>;
}

export function useAuth(): AuthContextValue {
  const ctx = use(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
