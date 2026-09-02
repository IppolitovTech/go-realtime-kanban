import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCorners,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { SortableContext, arrayMove, horizontalListSortingStrategy, sortableKeyboardCoordinates } from "@dnd-kit/sortable";
import { useCallback, useEffect, useRef, useState, type SubmitEvent } from "react";
import { api, errorMessage } from "../api/client";
import { connectBoardSocket, type BoardEvent } from "../api/socket";
import type { BoardDetail, Card, Column, ColumnDetail } from "../api/types";
import { ColumnView } from "./ColumnView";

// Replaces any existing entry with the same id (or appends) and keeps the
// array sorted by order_num. Used everywhere a column/card list is updated
// — from realtime events and from this tab's own REST responses alike —
// so the two paths can't drift into different orderings of the same data.
function upsertByOrder<T extends { id: string; order_num: number }>(items: T[], item: T): T[] {
  const next = [...items.filter((existing) => existing.id !== item.id), item];
  next.sort((a, b) => a.order_num - b.order_num);
  return next;
}

// Applies one realtime event (see docs/ru/websocket-events.md) to the
// current columns state. Every case except *.deleted upserts the version
// from the event, so the same handling covers "created", "updated" and
// "moved" uniformly — including a card that moved into a different column
// — and replaying an event this tab already applied via its own REST
// response is a harmless no-op (same data goes back in).
function applyBoardEvent(columns: ColumnDetail[], event: BoardEvent): ColumnDetail[] {
  switch (event.type) {
    case "column.deleted": {
      const column = event.data as Column;
      return columns.filter((c) => c.id !== column.id);
    }
    case "column.created":
    case "column.updated":
    case "column.moved": {
      const column = event.data as Column;
      const existingCards = columns.find((c) => c.id === column.id)?.cards ?? [];
      return upsertByOrder(columns, { ...column, cards: existingCards });
    }
    case "card.deleted": {
      const card = event.data as Card;
      return columns.map((c) => ({ ...c, cards: c.cards.filter((existing) => existing.id !== card.id) }));
    }
    case "card.created":
    case "card.updated":
    case "card.moved": {
      const card = event.data as Card;
      return columns.map((c) => {
        if (c.id !== card.column_id) {
          return c.cards.some((existing) => existing.id === card.id)
            ? { ...c, cards: c.cards.filter((existing) => existing.id !== card.id) }
            : c;
        }
        return { ...c, cards: upsertByOrder(c.cards, card) };
      });
    }
    default:
      return columns;
  }
}

interface Props {
  boardId: string;
  onBack: () => void;
}

export function BoardView({ boardId, onBack }: Props) {
  const [board, setBoard] = useState<BoardDetail | null>(null);
  const [columns, setColumns] = useState<ColumnDetail[]>([]);
  const [newColumnTitle, setNewColumnTitle] = useState("");
  const [error, setError] = useState<string | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const loadBoard = useCallback(() => {
    api
      .getBoard(boardId)
      .then((detail) => {
        setBoard(detail);
        setColumns(detail.columns);
      })
      .catch((err) => setError(errorMessage(err, "Failed to load board")));
  }, [boardId]);

  useEffect(() => {
    loadBoard();
  }, [loadBoard]);

  useEffect(() => {
    return connectBoardSocket(
      boardId,
      (event) => setColumns((prev) => applyBoardEvent(prev, event)),
      loadBoard,
    );
  }, [boardId, loadBoard]);

  async function handleAddColumn(e: SubmitEvent) {
    e.preventDefault();
    const title = newColumnTitle.trim();
    if (!title) return;
    try {
      const column = await api.createColumn(boardId, title);
      // upsertByOrder rather than blindly appending: the WS column.created
      // echo for this same creation can win the race and already be
      // applied (via applyBoardEvent) by the time this REST response
      // resolves — see websocket-events.md on why echoes aren't
      // suppressed, and why every insertion path needs to tolerate them.
      setColumns((prev) => upsertByOrder(prev, { ...column, cards: [] }));
      setNewColumnTitle("");
    } catch (err) {
      setError(errorMessage(err, "Failed to create column"));
    }
  }

  async function handleDeleteColumn(columnId: string) {
    try {
      await api.deleteColumn(columnId);
      setColumns((prev) => prev.filter((c) => c.id !== columnId));
    } catch (err) {
      setError(errorMessage(err, "Failed to delete column"));
    }
  }

  async function handleAddCard(columnId: string, title: string) {
    try {
      const card = await api.createCard(columnId, title, "");
      // upsertByOrder — see the matching comment in handleAddColumn.
      setColumns((prev) => prev.map((c) => (c.id === columnId ? { ...c, cards: upsertByOrder(c.cards, card) } : c)));
    } catch (err) {
      setError(errorMessage(err, "Failed to create card"));
    }
  }

  async function handleDeleteCard(cardId: string) {
    try {
      await api.deleteCard(cardId);
      setColumns((prev) => prev.map((c) => ({ ...c, cards: c.cards.filter((card) => card.id !== cardId) })));
    } catch (err) {
      setError(errorMessage(err, "Failed to delete card"));
    }
  }

  const dragOrigin = useRef<{ columnId: string; index: number } | null>(null);

  function handleDragStart(event: DragStartEvent) {
    setError(null);
    const activeId = event.active.id as string;
    const sourceCol = columns.find((c) => c.cards.some((card) => card.id === activeId));
    dragOrigin.current = sourceCol
      ? { columnId: sourceCol.id, index: sourceCol.cards.findIndex((c) => c.id === activeId) }
      : null;
  }

  function handleDragCancel() {
    dragOrigin.current = null;
    loadBoard();
  }

  // Live-reparents a dragged card into the column it's currently hovering
  // over, so the drop position looks right as you drag across columns.
  // Same-column reordering is deferred to handleDragEnd (arrayMove there
  // covers it) since dnd-kit fires many DragOver events during a single
  // drag and we only want one committed reorder at drop time.
  function handleDragOver(event: DragOverEvent) {
    const { active, over } = event;
    if (!over || active.data.current?.type !== "card") return;

    const activeId = active.id as string;
    const overId = over.id as string;
    if (activeId === overId) return;

    setColumns((prev) => {
      const sourceCol = prev.find((c) => c.cards.some((card) => card.id === activeId));
      const destCol = prev.find((c) => c.id === overId) ?? prev.find((c) => c.cards.some((card) => card.id === overId));
      if (!sourceCol || !destCol || sourceCol.id === destCol.id) return prev;

      const activeCard = sourceCol.cards.find((c) => c.id === activeId);
      if (!activeCard) return prev;
      const overCardIndex = destCol.cards.findIndex((c) => c.id === overId);
      const insertIndex = overCardIndex >= 0 ? overCardIndex : destCol.cards.length;

      return prev.map((col) => {
        if (col.id === sourceCol.id) {
          return { ...col, cards: col.cards.filter((c) => c.id !== activeId) };
        }
        if (col.id === destCol.id) {
          const newCards = [...col.cards];
          newCards.splice(insertIndex, 0, { ...activeCard, column_id: destCol.id });
          return { ...col, cards: newCards };
        }
        return col;
      });
    });
  }

  async function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    const origin = dragOrigin.current;
    dragOrigin.current = null;
    if (!over) {
      // handleDragOver may have already live-reparented the card into
      // another column's state; resync with the server since that change
      // was never persisted.
      loadBoard();
      return;
    }

    const activeId = active.id as string;
    const overId = over.id as string;

    if (active.data.current?.type === "column") {
      if (activeId === overId) return;

      // Reorder against the live `prev` (not the `columns` closure from
      // this render) so a WS event that landed mid-drag isn't clobbered —
      // and derive prevId/nextId from that same reordered array, so what's
      // sent to the API always matches what was actually applied.
      let prevId: string | null = null;
      let nextId: string | null = null;
      let moved = false;
      setColumns((prev) => {
        const oldIndex = prev.findIndex((c) => c.id === activeId);
        const newIndex = prev.findIndex((c) => c.id === overId);
        if (oldIndex === -1 || newIndex === -1) return prev;
        const next = arrayMove(prev, oldIndex, newIndex);
        const finalIndex = next.findIndex((c) => c.id === activeId);
        prevId = next[finalIndex - 1]?.id ?? null;
        nextId = next[finalIndex + 1]?.id ?? null;
        moved = true;
        return next;
      });
      if (!moved) return;

      try {
        await api.moveColumn(activeId, prevId, nextId);
      } catch (err) {
        setError(errorMessage(err, "Failed to move column"));
        loadBoard();
      }
      return;
    }

    const destColId = columns.find((c) => c.cards.some((c2) => c2.id === activeId))?.id;
    if (!destColId) return;

    // Same live-`prev` treatment as the column branch above.
    let prevCardId: string | null = null;
    let nextCardId: string | null = null;
    let finalIndex = -1;
    setColumns((prev) => {
      const col = prev.find((c) => c.id === destColId);
      if (!col) return prev;
      const oldIndex = col.cards.findIndex((c) => c.id === activeId);
      if (oldIndex === -1) return prev;
      const overIndex = col.cards.findIndex((c) => c.id === overId);
      const newCards = overIndex >= 0 && overIndex !== oldIndex ? arrayMove(col.cards, oldIndex, overIndex) : col.cards;
      finalIndex = newCards.findIndex((c) => c.id === activeId);
      prevCardId = newCards[finalIndex - 1]?.id ?? null;
      nextCardId = newCards[finalIndex + 1]?.id ?? null;
      return newCards === col.cards ? prev : prev.map((c) => (c.id === destColId ? { ...c, cards: newCards } : c));
    });
    if (finalIndex === -1) return;

    if (origin && origin.columnId === destColId && origin.index === finalIndex) {
      // Card ended up back where it started (e.g. dropped in place, or
      // dragged out and back) — nothing changed, skip the network call.
      return;
    }

    try {
      const updated = await api.moveCard(activeId, destColId, prevCardId, nextCardId);
      setColumns((prev) =>
        prev.map((c) => (c.id === destColId ? { ...c, cards: c.cards.map((card) => (card.id === activeId ? updated : card)) } : c)),
      );
    } catch (err) {
      setError(errorMessage(err, "Failed to move card"));
      loadBoard();
    }
  }

  if (!board) return <p className="p-6 text-center text-zinc-600 dark:text-zinc-300">Loading board…</p>;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex items-center gap-4 border-b border-zinc-200 px-6 py-4 dark:border-zinc-700">
        <button
          type="button"
          className="cursor-pointer rounded-md border border-zinc-200 bg-white px-3 py-1.5 text-sm text-zinc-950 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
          onClick={onBack}
        >
          ← Boards
        </button>
        <h1 className="m-0 font-semibold text-zinc-950 dark:text-zinc-100">{board.title}</h1>
      </header>

      {error && (
        <p className="mx-6 my-3 rounded-md bg-red-700/[0.12] px-3 py-2 text-red-700">{error}</p>
      )}

      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        // Auto-scroll is disabled: handleDragOver live-reparents the
        // dragged card into whichever column it's hovering (see below),
        // which changes DOM layout on every drag-over event. dnd-kit's
        // auto-scroll re-measures scrollable ancestors on that same
        // cadence, so the two feed each other and the affected container
        // (column list, or the board's horizontal row) scrolls in a
        // jittery loop near any edge instead of settling.
        autoScroll={false}
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragEnd={handleDragEnd}
        onDragCancel={handleDragCancel}
      >
        <SortableContext items={columns.map((c) => c.id)} strategy={horizontalListSortingStrategy}>
          <div className="flex max-h-full items-stretch gap-4 overflow-x-auto p-6">
            {columns.map((column) => (
              <ColumnView
                key={column.id}
                column={column}
                onAddCard={handleAddCard}
                onDeleteCard={handleDeleteCard}
                onDeleteColumn={handleDeleteColumn}
              />
            ))}

            <form
              className="m-0 flex items-center gap-2 self-start rounded-[10px] border border-zinc-200 bg-white p-1 dark:border-zinc-700 dark:bg-zinc-800"
              onSubmit={handleAddColumn}
            >
              <input
                type="text"
                placeholder="New column title"
                value={newColumnTitle}
                onChange={(e) => setNewColumnTitle(e.target.value)}
                maxLength={50}
                className="flex-1 rounded-md border border-zinc-200 bg-white px-2.5 py-2 text-zinc-950 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
              />
              <button
                type="submit"
                className="cursor-pointer rounded-md bg-violet-600 px-3.5 py-2 text-sm text-white hover:bg-violet-700 dark:bg-violet-400 dark:hover:bg-violet-300"
              >
                Add column
              </button>
            </form>
          </div>
        </SortableContext>
      </DndContext>
    </div>
  );
}
