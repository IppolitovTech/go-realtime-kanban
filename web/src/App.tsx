import { useState } from "react";
import { BoardList } from "./components/BoardList";
import { BoardView } from "./components/BoardView";

function App() {
  const [selectedBoardId, setSelectedBoardId] = useState<string | null>(null);

  if (selectedBoardId) {
    return <BoardView boardId={selectedBoardId} onBack={() => setSelectedBoardId(null)} />;
  }
  return <BoardList onOpenBoard={setSelectedBoardId} />;
}

export default App;
