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
import type { BoardDetail, ColumnDetail } from "../api/types";
import { ColumnView } from "./ColumnView";

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

  async function handleAddColumn(e: SubmitEvent) {
    e.preventDefault();
    const title = newColumnTitle.trim();
    if (!title) return;
    try {
      const column = await api.createColumn(boardId, title);
      setColumns((prev) => [...prev, { ...column, cards: [] }]);
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
      setColumns((prev) => prev.map((c) => (c.id === columnId ? { ...c, cards: [...c.cards, card] } : c)));
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
      const oldIndex = columns.findIndex((c) => c.id === activeId);
      const newIndex = columns.findIndex((c) => c.id === overId);
      if (oldIndex === -1 || newIndex === -1) return;

      const reordered = arrayMove(columns, oldIndex, newIndex);
      setColumns(reordered);
      const prevId = reordered[newIndex - 1]?.id ?? null;
      const nextId = reordered[newIndex + 1]?.id ?? null;
      try {
        await api.moveColumn(activeId, prevId, nextId);
      } catch (err) {
        setError(errorMessage(err, "Failed to move column"));
        loadBoard();
      }
      return;
    }

    const destCol = columns.find((c) => c.cards.some((c2) => c2.id === activeId));
    if (!destCol) return;

    const activeIndex = destCol.cards.findIndex((c) => c.id === activeId);
    const overCardIndex = destCol.cards.findIndex((c) => c.id === overId);
    let finalCards = destCol.cards;
    if (overCardIndex >= 0 && overCardIndex !== activeIndex) {
      finalCards = arrayMove(destCol.cards, activeIndex, overCardIndex);
      setColumns((prev) => prev.map((c) => (c.id === destCol.id ? { ...c, cards: finalCards } : c)));
    }

    const finalIndex = finalCards.findIndex((c) => c.id === activeId);
    if (origin && origin.columnId === destCol.id && origin.index === finalIndex) {
      // Card ended up back where it started (e.g. dropped in place, or
      // dragged out and back) — nothing changed, skip the network call.
      return;
    }

    const prevCardId = finalCards[finalIndex - 1]?.id ?? null;
    const nextCardId = finalCards[finalIndex + 1]?.id ?? null;

    try {
      const updated = await api.moveCard(activeId, destCol.id, prevCardId, nextCardId);
      setColumns((prev) =>
        prev.map((c) => (c.id === destCol.id ? { ...c, cards: c.cards.map((card) => (card.id === activeId ? updated : card)) } : c)),
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
