import { useState } from "react";
import { useAuth } from "./auth/AuthContext";
import { BoardList } from "./components/BoardList";
import { BoardView } from "./components/BoardView";
import { LoginScreen } from "./components/LoginScreen";

function App() {
  const { user, logout } = useAuth();
  const [selectedBoardId, setSelectedBoardId] = useState<string | null>(null);

  if (!user) {
    return <LoginScreen />;
  }

  // selectedBoardId lives in this component, which never unmounts across a
  // logout/login cycle — clear it here so a fresh login (possibly as a
  // different user, who may not be a member of the previously open board)
  // always lands back on the board list rather than that stale board view.
  function handleLogout() {
    setSelectedBoardId(null);
    logout();
  }

  return (
    <div className="flex min-h-screen flex-col">
      <header className="flex items-center justify-between border-b border-zinc-200 px-6 py-2 dark:border-zinc-700">
        <span className="text-sm text-zinc-600 dark:text-zinc-300">{user.name}</span>
        <button
          type="button"
          onClick={handleLogout}
          className="cursor-pointer bg-transparent text-sm text-violet-600 underline dark:text-violet-400"
        >
          Log out
        </button>
      </header>
      {selectedBoardId ? (
        <BoardView boardId={selectedBoardId} onBack={() => setSelectedBoardId(null)} />
      ) : (
        <BoardList onOpenBoard={setSelectedBoardId} />
      )}
    </div>
  );
}

export default App;
