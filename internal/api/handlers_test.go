package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"db-design-project/internal/api"
	"db-design-project/internal/db"
)

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "unit-test-secret-key")
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// newRouter builds a gin engine with routes registered against q.
func newRouter(q db.Querier) *gin.Engine {
	r := gin.New()
	api.RegisterRoutes(r, q)
	return r
}

// makeToken returns a valid Bearer token for the given userID signed with the test secret.
func makeToken(userID int32) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": float64(userID),
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	signed, _ := token.SignedString([]byte("unit-test-secret-key"))
	return "Bearer " + signed
}

func hashPassword(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

// ── POST /login ────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	hash := hashPassword(t, "secret")
	q := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, email string) (db.User, error) {
			return db.User{ID: 1, Email: email, PasswordHash: hash}, nil
		},
	}
	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "password": "secret"})
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["access_token"] == "" {
		t.Fatal("expected access_token in response")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hash := hashPassword(t, "correct")
	q := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (db.User, error) {
			return db.User{ID: 1, PasswordHash: hash}, nil
		},
	}
	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "password": "wrong"})
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	q := &mockQuerier{
		getUserByEmailFn: func(_ context.Context, _ string) (db.User, error) {
			return db.User{}, sql.ErrNoRows
		},
	}
	body, _ := json.Marshal(map[string]string{"email": "nobody@b.com", "password": "x"})
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogin_MalformedBody(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("not json")))
	r.Header.Set("Content-Type", "application/json")

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── GET /products ──────────────────────────────────────────────────────────

func TestListProducts_Success(t *testing.T) {
	q := &mockQuerier{
		listAvailableProductsFn: func(_ context.Context, _ db.ListAvailableProductsParams) ([]db.Product, error) {
			return []db.Product{
				{ID: 1, Sku: "SKU-1", Name: "Widget", Price: "9.99"},
			}, nil
		},
	}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/products", nil)

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var products []db.Product
	json.NewDecoder(w.Body).Decode(&products)
	if len(products) != 1 || products[0].Sku != "SKU-1" {
		t.Fatalf("unexpected products: %v", products)
	}
}

func TestListProducts_DBError(t *testing.T) {
	q := &mockQuerier{
		listAvailableProductsFn: func(_ context.Context, _ db.ListAvailableProductsParams) ([]db.Product, error) {
			return nil, fmt.Errorf("connection reset")
		},
	}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/products", nil)

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── GET /products/:id ──────────────────────────────────────────────────────

func TestGetProduct_Found(t *testing.T) {
	q := &mockQuerier{
		getProductByIDFn: func(_ context.Context, id int32) (db.Product, error) {
			return db.Product{ID: id, Sku: "SKU-42", Name: "Gadget", Price: "19.99"}, nil
		},
	}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/products/42", nil)

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	q := &mockQuerier{
		getProductByIDFn: func(_ context.Context, _ int32) (db.Product, error) {
			return db.Product{}, sql.ErrNoRows
		},
	}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/products/999", nil)

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ── POST /orders ───────────────────────────────────────────────────────────

func TestCreateOrder_Authenticated(t *testing.T) {
	q := &mockQuerier{
		createOrderFn: func(_ context.Context, arg db.CreateOrderParams) (db.Order, error) {
			return db.Order{ID: 1, UserID: arg.UserID, Total: arg.Total}, nil
		},
	}
	body, _ := json.Marshal(map[string]float64{"total": 49.99})
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", makeToken(7))

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateOrder_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(`{"total":10}`)))
	r.Header.Set("Content-Type", "application/json")
	// No Authorization header

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateOrder_MalformedBody(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte("not json")))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", makeToken(1))

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── GET /orders ────────────────────────────────────────────────────────────

func TestListOrders_Authenticated(t *testing.T) {
	q := &mockQuerier{
		listOrdersByUserFn: func(_ context.Context, userID int32) ([]db.Order, error) {
			return []db.Order{{ID: 10, UserID: userID}}, nil
		},
	}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/orders", nil)
	r.Header.Set("Authorization", makeToken(3))

	newRouter(q).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListOrders_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/orders", nil)

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── GET /users/me ──────────────────────────────────────────────────────────

func TestGetMe_Authenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/users/me", nil)
	r.Header.Set("Authorization", makeToken(5))

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if int(resp["user_id"].(float64)) != 5 {
		t.Fatalf("expected user_id=5, got %v", resp["user_id"])
	}
}

func TestGetMe_Unauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/users/me", nil)

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
