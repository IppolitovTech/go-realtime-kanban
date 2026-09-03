// Tiny localStorage wrapper for the JWT — see docs/ru/adr/005-jwt-vs-sessions.md
// for why localStorage (no state library in this app) rather than a
// cookie. Kept separate from AuthContext so api/client.ts and api/socket.ts
// can read the token without importing React.
import type { User } from "../api/types";

const TOKEN_KEY = "kanban_token";
const USER_KEY = "kanban_user";

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function getStoredUser(): User | null {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as User;
  } catch {
    return null;
  }
}

export function setSession(token: string, user: User): void {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}
