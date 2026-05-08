package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// newMiddlewareRouter builds a minimal router that uses AuthMiddleware and
// returns 200 with the resolved user_id if authentication passes.
func newMiddlewareRouter() *gin.Engine {
	r := gin.New()
	r.GET("/protected", authTestHandler())
	return r
}

func authTestHandler() gin.HandlerFunc {
	// Import AuthMiddleware via RegisterRoutes — easier to test through the
	// full router since AuthMiddleware is unexported.
	// We use newRouter with a mockQuerier and hit a known auth-protected route.
	return func(c *gin.Context) {
		// This handler is never reached directly; see tests below.
		c.Status(http.StatusOK)
	}
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/users/me", nil)
	// No Authorization header at all

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongScheme(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/users/me", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Basic, not Bearer

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_MalformedToken(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/users/me", nil)
	r.Header.Set("Authorization", "Bearer not.a.jwt")

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": float64(1),
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // already expired
	})
	signed, _ := token.SignedString([]byte("unit-test-secret-key"))

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/users/me", nil)
	r.Header.Set("Authorization", "Bearer "+signed)

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": float64(1),
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	signed, _ := token.SignedString([]byte("wrong-secret"))

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/users/me", nil)
	r.Header.Set("Authorization", "Bearer "+signed)

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong secret, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/users/me", nil)
	r.Header.Set("Authorization", makeToken(99))

	newRouter(&mockQuerier{}).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token, got %d", w.Code)
	}
}
