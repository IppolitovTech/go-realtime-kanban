import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { Card } from "../api/types";

interface Props {
  card: Card;
  onDelete: (cardId: string) => void;
}

export function CardItem({ card, onDelete }: Props) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: card.id,
    data: { type: "card", columnId: card.column_id },
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="relative cursor-grab rounded-lg border border-zinc-200 bg-zinc-100 py-2.5 pr-7 pl-3 text-left dark:border-zinc-700 dark:bg-zinc-900"
      {...attributes}
      {...listeners}
    >
      <div className="font-medium text-zinc-950 dark:text-zinc-100">{card.title}</div>
      {card.description && (
        <div className="mt-1 text-[13px] text-zinc-600 dark:text-zinc-300">{card.description}</div>
      )}
      <button
        type="button"
        className="absolute top-2 right-2 h-5 w-5 cursor-pointer rounded text-base leading-none text-zinc-600 hover:text-red-700 dark:text-zinc-300"
        onPointerDown={(e) => e.stopPropagation()}
        onClick={() => onDelete(card.id)}
        aria-label="Delete card"
      >
        ×
      </button>
    </div>
  );
}
