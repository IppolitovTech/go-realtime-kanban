import type { Board, BoardDetail, BoardMember, Card, Column } from "./types";

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1";

// No X-User-ID header is sent: Stage 1 has no registration yet (see
// architecture.md, "Заглушка пользователя на Этапе 1"), and the API
// already falls back to its seed user when the header is absent. Stage 2
// swaps this client for one that attaches a real bearer token — nothing
// else here changes.
export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export function errorMessage(err: unknown, fallback: string): string {
  return err instanceof ApiError ? err.message : fallback;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: { "Content-Type": "application/json", ...options.headers },
  });
  if (!res.ok) {
    const body: { error?: string } = await res.json().catch(() => ({}));
    throw new ApiError(res.status, body.error ?? res.statusText);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export const api = {
  listBoards: () => request<Board[]>("/boards"),
  createBoard: (title: string) => request<Board>("/boards", { method: "POST", body: JSON.stringify({ title }) }),
  getBoard: (id: string) => request<BoardDetail>(`/boards/${id}`),
  updateBoard: (id: string, title: string) =>
    request<Board>(`/boards/${id}`, { method: "PATCH", body: JSON.stringify({ title }) }),
  deleteBoard: (id: string) => request<void>(`/boards/${id}`, { method: "DELETE" }),
  inviteMember: (boardId: string, email: string) =>
    request<BoardMember>(`/boards/${boardId}/members`, { method: "POST", body: JSON.stringify({ email }) }),

  createColumn: (boardId: string, title: string) =>
    request<Column>(`/boards/${boardId}/columns`, { method: "POST", body: JSON.stringify({ title }) }),
  updateColumn: (id: string, title: string) =>
    request<Column>(`/columns/${id}`, { method: "PATCH", body: JSON.stringify({ title }) }),
  deleteColumn: (id: string) => request<void>(`/columns/${id}`, { method: "DELETE" }),
  moveColumn: (id: string, prevColumnId: string | null, nextColumnId: string | null) =>
    request<Column>(`/columns/${id}/move`, {
      method: "PATCH",
      body: JSON.stringify({ prev_column_id: prevColumnId, next_column_id: nextColumnId }),
    }),

  createCard: (columnId: string, title: string, description: string) =>
    request<Card>(`/columns/${columnId}/cards`, { method: "POST", body: JSON.stringify({ title, description }) }),
  updateCard: (id: string, patch: { title?: string; description?: string }) =>
    request<Card>(`/cards/${id}`, { method: "PATCH", body: JSON.stringify(patch) }),
  deleteCard: (id: string) => request<void>(`/cards/${id}`, { method: "DELETE" }),
  moveCard: (id: string, targetColumnId: string, prevCardId: string | null, nextCardId: string | null) =>
    request<Card>(`/cards/${id}/move`, {
      method: "PATCH",
      body: JSON.stringify({ target_column_id: targetColumnId, prev_card_id: prevCardId, next_card_id: nextCardId }),
    }),
};
