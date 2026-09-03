package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
	"github.com/IppolitovTech/go-realtime-kanban/internal/repository"
	"github.com/IppolitovTech/go-realtime-kanban/internal/service"
	transporthttp "github.com/IppolitovTech/go-realtime-kanban/internal/transport/http"
)

// stubUserRepo is a minimal repository.UserRepository, local to this
// package's tests (the shared in-memory mocks in internal/service are
// package-private to internal/service and not reusable here).
type stubUserRepo struct {
	mu    sync.Mutex
	users map[uuid.UUID]domain.User
}

var _ repository.UserRepository = (*stubUserRepo)(nil)

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{users: map[uuid.UUID]domain.User{}}
}

func (r *stubUserRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *stubUserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (r *stubUserRepo) Create(ctx context.Context, user domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == user.Email {
			return domain.User{}, domain.ErrEmailTaken
		}
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	r.users[user.ID] = user
	return user, nil
}

func newAuthHandlerForTest() *transporthttp.AuthHandler {
	authService := service.NewAuthService(newStubUserRepo(), []byte("test-secret"), time.Hour)
	return transporthttp.NewAuthHandler(authService)
}

func doJSON(t *testing.T, body any) *bytes.Reader {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return bytes.NewReader(raw)
}

func TestAuthHandler_RegisterThenLogin(t *testing.T) {
	handler := newAuthHandlerForTest()

	registerReq := httptest.NewRequest("POST", "/auth/register", doJSON(t, map[string]string{
		"email":    "alice@example.com",
		"password": "password123",
		"name":     "Alice",
	}))
	registerRec := httptest.NewRecorder()
	handler.Register(registerRec, registerReq)

	if registerRec.Code != 201 {
		t.Fatalf("Register() status = %d, want 201; body = %s", registerRec.Code, registerRec.Body)
	}
	var registerBody struct {
		Token string `json:"token"`
		User  struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(registerRec.Body.Bytes(), &registerBody); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}
	if registerBody.Token == "" || registerBody.User.Email != "alice@example.com" {
		t.Fatalf("Register() body = %+v", registerBody)
	}

	loginReq := httptest.NewRequest("POST", "/auth/login", doJSON(t, map[string]string{
		"email":    "alice@example.com",
		"password": "password123",
	}))
	loginRec := httptest.NewRecorder()
	handler.Login(loginRec, loginReq)

	if loginRec.Code != 200 {
		t.Fatalf("Login() status = %d, want 200; body = %s", loginRec.Code, loginRec.Body)
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	handler := newAuthHandlerForTest()
	body := map[string]string{"email": "alice@example.com", "password": "password123", "name": "Alice"}

	first := httptest.NewRecorder()
	handler.Register(first, httptest.NewRequest("POST", "/auth/register", doJSON(t, body)))
	if first.Code != 201 {
		t.Fatalf("first Register() status = %d, want 201", first.Code)
	}

	second := httptest.NewRecorder()
	handler.Register(second, httptest.NewRequest("POST", "/auth/register", doJSON(t, body)))
	if second.Code != 409 {
		t.Fatalf("second Register() status = %d, want 409; body = %s", second.Code, second.Body)
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	handler := newAuthHandlerForTest()
	handler.Register(httptest.NewRecorder(), httptest.NewRequest("POST", "/auth/register", doJSON(t, map[string]string{
		"email": "alice@example.com", "password": "password123", "name": "Alice",
	})))

	rec := httptest.NewRecorder()
	handler.Login(rec, httptest.NewRequest("POST", "/auth/login", doJSON(t, map[string]string{
		"email": "alice@example.com", "password": "wrong-password",
	})))
	if rec.Code != 401 {
		t.Fatalf("Login() status = %d, want 401; body = %s", rec.Code, rec.Body)
	}
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	handler := newAuthHandlerForTest()
	rec := httptest.NewRecorder()
	handler.Register(rec, httptest.NewRequest("POST", "/auth/register", doJSON(t, map[string]string{
		"email": "not-an-email", "password": "password123", "name": "Alice",
	})))
	if rec.Code != 400 {
		t.Fatalf("Register() status = %d, want 400; body = %s", rec.Code, rec.Body)
	}
}
