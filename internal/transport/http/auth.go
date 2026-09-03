package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
)

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt string    `json:"created_at"`
}

func newUserResponse(u domain.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt.Format(timeFormat),
	}
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthHandler exposes /auth/register and /auth/login — see
// docs/ru/openapi.yaml's RegisterRequest/LoginRequest/AuthResponse
// schemas. Unlike every other handler, these two sit outside the JWT
// middleware group (main.go) since there's no token yet to verify.
type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, token, err := h.auth.Register(r.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, authResponse{Token: token, User: newUserResponse(user)})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		RespondDomainError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, authResponse{Token: token, User: newUserResponse(user)})
}
