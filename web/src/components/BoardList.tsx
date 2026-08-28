import { useEffect, useState, type SubmitEvent } from "react";
import { api, errorMessage } from "../api/client";
import type { Board } from "../api/types";

interface Props {
  onOpenBoard: (boardId: string) => void;
}

export function BoardList({ onOpenBoard }: Props) {
  const [boards, setBoards] = useState<Board[]>([]);
  const [newTitle, setNewTitle] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .listBoards()
      .then((data) => {
        if (!cancelled) setBoards(data);
      })
      .catch((err) => {
        if (!cancelled) setError(errorMessage(err, "Failed to load boards"));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleCreate(e: SubmitEvent) {
    e.preventDefault();
    const title = newTitle.trim();
    if (!title) return;
    setError(null);
    try {
      const board = await api.createBoard(title);
      setBoards((prev) => [...prev, board]);
      setNewTitle("");
    } catch (err) {
      setError(errorMessage(err, "Failed to create board"));
    }
  }

  if (loading) return <p className="p-6 text-center text-zinc-600 dark:text-zinc-300">Loading boards…</p>;

  return (
    <div className="mx-auto w-full max-w-[640px] px-4 py-8">
      <h1 className="mt-0 mb-4 font-semibold text-zinc-950 dark:text-zinc-100">Your boards</h1>
      {error && (
        <p className="mx-6 my-3 rounded-md bg-red-700/[0.12] px-3 py-2 text-red-700">{error}</p>
      )}

      <form className="flex gap-2" onSubmit={handleCreate}>
        <input
          type="text"
          placeholder="New board title"
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          maxLength={100}
          className="flex-1 rounded-md border border-zinc-200 bg-white px-2.5 py-2 text-zinc-950 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
        />
        <button
          type="submit"
          className="cursor-pointer rounded-md bg-violet-600 px-3.5 py-2 text-sm text-white hover:bg-violet-700 dark:bg-violet-400 dark:hover:bg-violet-300"
        >
          Create board
        </button>
      </form>

      {boards.length === 0 ? (
        <p className="p-6 text-center text-zinc-600 dark:text-zinc-300">No boards yet — create one above.</p>
      ) : (
        <ul className="mt-2 mb-0 flex list-none flex-col gap-2 p-0">
          {boards.map((board) => (
            <li key={board.id}>
              <button
                className="w-full cursor-pointer rounded-lg border border-zinc-200 bg-white px-4 py-3.5 text-left text-sm text-zinc-950 hover:border-violet-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100 dark:hover:border-violet-400"
                onClick={() => onOpenBoard(board.id)}
              >
                {board.title}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
