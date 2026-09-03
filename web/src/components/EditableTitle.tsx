import { useEffect, useRef, useState } from "react";

interface Props {
  value: string;
  onSave: (newValue: string) => void;
  maxLength?: number;
  className?: string;
  // Rendered as this tag while not editing. Kept a plain string (not JSX)
  // so callers can put an <h1> in the board header and a <span> in a
  // column/card header from the same component.
  as?: "span" | "h1" | "div";
}

// Double-click a title to rename it in place; Enter/blur saves, Escape
// reverts. `draft` is null while not editing, so it also serves as the
// edit-mode flag — no separate boolean to keep in sync with it. The input
// stops pointer/keyboard events from bubbling up, since a title can sit
// inside a larger clickable/draggable area (e.g. a drag handle) that would
// otherwise treat typing or clicking as its own gesture.
export function EditableTitle({ value, onSave, maxLength, className, as: As = "span" }: Props) {
  const [draft, setDraft] = useState<string | null>(null);
  const editing = draft !== null;
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }, [editing]);

  function commit() {
    const trimmed = draft?.trim();
    setDraft(null);
    if (trimmed && trimmed !== value) onSave(trimmed);
  }

  if (draft !== null) {
    return (
      <input
        ref={inputRef}
        type="text"
        value={draft}
        maxLength={maxLength}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onPointerDown={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          e.stopPropagation();
          if (e.key === "Enter") {
            e.preventDefault();
            commit();
          } else if (e.key === "Escape") {
            e.preventDefault();
            setDraft(null);
          }
        }}
        className={`${className ?? ""} w-full rounded-md border border-violet-600 bg-white px-1 text-inherit dark:bg-zinc-800`}
      />
    );
  }

  return (
    <As className={className} title="Double-click to rename" onDoubleClick={() => setDraft(value)}>
      {value}
    </As>
  );
}
