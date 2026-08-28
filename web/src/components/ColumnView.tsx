import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useState, type SubmitEvent } from "react";
import type { ColumnDetail } from "../api/types";
import { CardItem } from "./CardItem";

interface Props {
  column: ColumnDetail;
  onAddCard: (columnId: string, title: string) => void;
  onDeleteCard: (cardId: string) => void;
  onDeleteColumn: (columnId: string) => void;
}

export function ColumnView({ column, onAddCard, onDeleteCard, onDeleteColumn }: Props) {
  const [newCardTitle, setNewCardTitle] = useState("");
  const { setNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
    id: column.id,
    data: { type: "column", columnId: column.id },
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  function handleAddCard(e: SubmitEvent) {
    e.preventDefault();
    const title = newCardTitle.trim();
    if (!title) return;
    onAddCard(column.id, title);
    setNewCardTitle("");
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex max-h-full w-[280px] flex-none flex-col rounded-[10px] border border-zinc-200 bg-white dark:border-zinc-700 dark:bg-zinc-800"
    >
      <div
        className="flex cursor-grab items-center justify-between border-b border-zinc-200 px-3.5 py-3 font-semibold text-zinc-950 dark:border-zinc-700 dark:text-zinc-100"
        {...attributes}
        {...listeners}
      >
        <span>{column.title}</span>
        <button
          type="button"
          className="h-5 w-5 cursor-pointer rounded text-base leading-none text-zinc-600 hover:text-red-700 dark:text-zinc-300"
          onPointerDown={(e) => e.stopPropagation()}
          onClick={() => onDeleteColumn(column.id)}
          aria-label="Delete column"
        >
          ×
        </button>
      </div>

      <SortableContext items={column.cards.map((c) => c.id)} strategy={verticalListSortingStrategy}>
        <div className="flex min-h-[40px] flex-1 flex-col gap-2 overflow-y-auto p-2.5">
          {column.cards.map((card) => (
            <CardItem key={card.id} card={card} onDelete={onDeleteCard} />
          ))}
        </div>
      </SortableContext>

      <form className="m-0 flex gap-2 p-2.5" onSubmit={handleAddCard}>
        <input
          type="text"
          placeholder="New card title"
          value={newCardTitle}
          onChange={(e) => setNewCardTitle(e.target.value)}
          maxLength={255}
          className="flex-1 rounded-md border border-zinc-200 bg-white px-2.5 py-2 text-zinc-950 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
        />
        <button
          type="submit"
          className="cursor-pointer rounded-md bg-violet-600 px-3.5 py-2 text-sm text-white hover:bg-violet-700 dark:bg-violet-400 dark:hover:bg-violet-300"
        >
          Add card
        </button>
      </form>
    </div>
  );
}
