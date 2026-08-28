package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
)

type boardResponse struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt string    `json:"created_at"`
}

func newBoardResponse(b domain.Board) boardResponse {
	return boardResponse{
		ID:        b.ID,
		Title:     b.Title,
		OwnerID:   b.OwnerID,
		CreatedAt: b.CreatedAt.Format(timeFormat),
	}
}

type boardMemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	JoinedAt string    `json:"joined_at"`
}

func newBoardMemberResponse(m domain.BoardMember) boardMemberResponse {
	return boardMemberResponse{
		UserID:   m.UserID,
		Email:    m.Email,
		Name:     m.Name,
		JoinedAt: m.JoinedAt.Format(timeFormat),
	}
}

type boardDetailResponse struct {
	ID        uuid.UUID              `json:"id"`
	Title     string                 `json:"title"`
	OwnerID   uuid.UUID              `json:"owner_id"`
	Columns   []columnDetailResponse `json:"columns"`
	Members   []boardMemberResponse  `json:"members"`
	CreatedAt string                 `json:"created_at"`
}

type createBoardRequest struct {
	Title string `json:"title"`
}

type updateBoardRequest struct {
	Title string `json:"title"`
}

type inviteMemberRequest struct {
	Email string `json:"email"`
}

// BoardHandler exposes the /boards* REST endpoints. It orchestrates calls
// across BoardService/ColumnService/CardService to assemble BoardDetail —
// that assembly is response-shaping, not business logic, so it lives here
// rather than in a service (see architecture.md on transport's role).
type BoardHandler struct {
	boards  *service.BoardService
	columns *service.ColumnService
	cards   *service.CardService
}

func NewBoardHandler(boards *service.BoardService, columns *service.ColumnService, cards *service.CardService) *BoardHandler {
	return &BoardHandler{boards: boards, columns: columns, cards: cards}
}

func (h *BoardHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	boards, err := h.boards.List(r.Context(), userID)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	out := make([]boardResponse, len(boards))
	for i, b := range boards {
		out[i] = newBoardResponse(b)
	}
	RespondJSON(w, http.StatusOK, out)
}

func (h *BoardHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	var req createBoardRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	board, err := h.boards.Create(r.Context(), userID, req.Title)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, newBoardResponse(board))
}

func (h *BoardHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	boardID, ok := parseUUIDParam(w, r, "boardId", "board id")
	if !ok {
		return
	}

	board, err := h.boards.Get(r.Context(), userID, boardID)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	columns, err := h.columns.ListByBoard(r.Context(), userID, boardID)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	members, err := h.boards.ListMembers(r.Context(), userID, boardID)
	if err != nil {
		RespondDomainError(w, err)
		return
	}

	columnDetails := make([]columnDetailResponse, len(columns))
	for i, c := range columns {
		cards, err := h.cards.ListByColumn(r.Context(), userID, c.ID)
		if err != nil {
			RespondDomainError(w, err)
			return
		}
		columnDetails[i] = newColumnDetailResponse(c, cards)
	}
	memberResponses := make([]boardMemberResponse, len(members))
	for i, m := range members {
		memberResponses[i] = newBoardMemberResponse(m)
	}

	RespondJSON(w, http.StatusOK, boardDetailResponse{
		ID:        board.ID,
		Title:     board.Title,
		OwnerID:   board.OwnerID,
		Columns:   columnDetails,
		Members:   memberResponses,
		CreatedAt: board.CreatedAt.Format(timeFormat),
	})
}

func (h *BoardHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	boardID, ok := parseUUIDParam(w, r, "boardId", "board id")
	if !ok {
		return
	}
	var req updateBoardRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	board, err := h.boards.UpdateTitle(r.Context(), userID, boardID, req.Title)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, newBoardResponse(board))
}

func (h *BoardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	boardID, ok := parseUUIDParam(w, r, "boardId", "board id")
	if !ok {
		return
	}

	if err := h.boards.Delete(r.Context(), userID, boardID); err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusNoContent, nil)
}

func (h *BoardHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	boardID, ok := parseUUIDParam(w, r, "boardId", "board id")
	if !ok {
		return
	}
	var req inviteMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	member, err := h.boards.InviteMember(r.Context(), userID, boardID, req.Email)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, newBoardMemberResponse(member))
}
