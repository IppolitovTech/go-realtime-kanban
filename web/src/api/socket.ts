// Realtime board events — see docs/ru/websocket-events.md for the wire
// format this mirrors.
import type { Card, Column } from "./types";

export type BoardEventType =
  | "column.created"
  | "column.updated"
  | "column.deleted"
  | "column.moved"
  | "card.created"
  | "card.updated"
  | "card.deleted"
  | "card.moved";

export interface BoardEvent {
  type: BoardEventType;
  board_id: string;
  data: Card | Column;
  occurred_at: string;
}

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1";
const INITIAL_RECONNECT_DELAY_MS = 2000;
const MAX_RECONNECT_DELAY_MS = 30000;

function boardSocketURL(boardId: string): string {
  const httpBase = new URL(API_BASE, window.location.href);
  const wsProtocol = httpBase.protocol === "https:" ? "wss:" : "ws:";
  return `${wsProtocol}//${httpBase.host}${httpBase.pathname.replace(/\/$/, "")}/boards/${boardId}/ws`;
}

// connectBoardSocket subscribes to boardId's realtime events and keeps
// reconnecting (after a fixed delay) if the connection drops, until the
// returned cleanup function is called. There's no event backlog on the
// server to replay on reconnect — see websocket-events.md's
// deliberately-out-of-scope section — so instead onReconnect is called
// every time a dropped connection comes back, and the caller re-reads the
// whole board to pick up anything it missed while disconnected.
export function connectBoardSocket(boardId: string, onEvent: (event: BoardEvent) => void, onReconnect: () => void): () => void {
  let socket: WebSocket | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let stopped = false;
  let hasConnectedBefore = false;
  // Doubles on every failed attempt (capped) and resets on a successful
  // connection — a permanently rejected handshake (board deleted, no
  // longer a member) still fires onclose just like a dropped connection,
  // so this keeps retrying it from hammering the server every 2s forever.
  let reconnectDelay = INITIAL_RECONNECT_DELAY_MS;

  function connect() {
    if (stopped) return;
    socket = new WebSocket(boardSocketURL(boardId));

    socket.onopen = () => {
      reconnectDelay = INITIAL_RECONNECT_DELAY_MS;
      if (hasConnectedBefore) onReconnect();
      hasConnectedBefore = true;
    };

    socket.onmessage = (e) => {
      try {
        onEvent(JSON.parse(e.data as string) as BoardEvent);
      } catch {
        // Malformed frame — drop it rather than take the socket down.
      }
    };

    socket.onclose = () => {
      if (stopped) return;
      reconnectTimer = setTimeout(connect, reconnectDelay);
      reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY_MS);
    };
  }

  connect();

  return () => {
    stopped = true;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    socket?.close();
  };
}
