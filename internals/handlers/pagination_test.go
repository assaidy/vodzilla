package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func TestEncodeDecodeCursor(t *testing.T) {
	t.Run("uuid round-trip", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		encoded := encodeCursor(id)
		decoded, err := decodeCursor[uuid.UUID](encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if decoded != id {
			t.Errorf("want %v, got %v", id, decoded)
		}
	})

	t.Run("int64 round-trip", func(t *testing.T) {
		v := int64(42)
		encoded := encodeCursor(v)
		decoded, err := decodeCursor[int64](encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if decoded != v {
			t.Errorf("want %d, got %d", v, decoded)
		}
	})

	t.Run("string round-trip", func(t *testing.T) {
		v := "hello"
		encoded := encodeCursor(v)
		decoded, err := decodeCursor[string](encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if decoded != v {
			t.Errorf("want %s, got %s", v, decoded)
		}
	})

	t.Run("empty string returns zero value", func(t *testing.T) {
		decoded, err := decodeCursor[uuid.UUID]("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if decoded != uuid.Nil {
			t.Errorf("want zero value, got %v", decoded)
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := decodeCursor[uuid.UUID]("!!!not-base64!!!")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if _, ok := errors.AsType[apiError](err); !ok {
			t.Fatalf("expected apiError, got %T", err)
		}
	})

	t.Run("valid base64 but invalid json", func(t *testing.T) {
		encoded := base64.URLEncoding.EncodeToString([]byte("not-json"))
		_, err := decodeCursor[uuid.UUID](encoded)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if _, ok := errors.AsType[apiError](err); !ok {
			t.Fatalf("expected apiError, got %T", err)
		}
	})

	t.Run("marshal failure returns empty", func(t *testing.T) {
		encoded := encodeCursor(make(chan int))
		if encoded != "" {
			t.Errorf("expected empty string, got %s", encoded)
		}
	})
}

func TestNewPaginatedResponse(t *testing.T) {
	t.Run("items less than limit", func(t *testing.T) {
		items := []int{1, 2, 3}
		resp := newPaginatedResponse(items, 10)
		if len(resp.Items) != 3 {
			t.Errorf("want 3 items, got %d", len(resp.Items))
		}
		if resp.HasMore {
			t.Error("HasMore should be false")
		}
		if resp.Cursor != "" {
			t.Error("Cursor should be empty")
		}
	})

	t.Run("items at limit", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		resp := newPaginatedResponse(items, 3)
		if !resp.HasMore {
			t.Error("HasMore should be true when len(items) == limit")
		}
	})

	t.Run("empty items", func(t *testing.T) {
		items := []int{}
		resp := newPaginatedResponse(items, 15)
		if len(resp.Items) != 0 {
			t.Errorf("want 0 items, got %d", len(resp.Items))
		}
		if resp.HasMore {
			t.Error("HasMore should be false for empty items")
		}
	})
}

func TestParsePaginatedRequest(t *testing.T) {
	app := fiber.New()

	t.Run("default limit", func(t *testing.T) {
		app.Get("/test-default", func(c fiber.Ctx) error {
			pr, err := parsePaginatedRequest[uuid.UUID](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pr.Limit != minPageLimit {
				t.Errorf("want limit %d, got %d", minPageLimit, pr.Limit)
			}
			if pr.Cursor != uuid.Nil {
				t.Errorf("want nil cursor, got %v", pr.Cursor)
			}
			return nil
		})
		req, _ := http.NewRequest(http.MethodGet, "/test-default", nil)
		resp, _ := app.Test(req)
		resp.Body.Close()
	})

	t.Run("custom limit", func(t *testing.T) {
		app.Get("/test-limit", func(c fiber.Ctx) error {
			pr, err := parsePaginatedRequest[uuid.UUID](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pr.Limit != 50 {
				t.Errorf("want limit 50, got %d", pr.Limit)
			}
			return nil
		})
		req, _ := http.NewRequest(http.MethodGet, "/test-limit?limit=50", nil)
		resp, _ := app.Test(req)
		resp.Body.Close()
	})

	t.Run("limit too low", func(t *testing.T) {
		app.Get("/test-low", func(c fiber.Ctx) error {
			_, err := parsePaginatedRequest[uuid.UUID](c)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if _, ok := errors.AsType[apiError](err); !ok {
				t.Fatalf("expected apiError, got %T", err)
			}
			return nil
		})
		req, _ := http.NewRequest(http.MethodGet, "/test-low?limit=1", nil)
		resp, _ := app.Test(req)
		resp.Body.Close()
	})

	t.Run("limit too high", func(t *testing.T) {
		app.Get("/test-high", func(c fiber.Ctx) error {
			_, err := parsePaginatedRequest[uuid.UUID](c)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if _, ok := errors.AsType[apiError](err); !ok {
				t.Fatalf("expected apiError, got %T", err)
			}
			return nil
		})
		req, _ := http.NewRequest(http.MethodGet, "/test-high?limit=200", nil)
		resp, _ := app.Test(req)
		resp.Body.Close()
	})

	t.Run("invalid limit string", func(t *testing.T) {
		app.Get("/test-inv", func(c fiber.Ctx) error {
			_, err := parsePaginatedRequest[uuid.UUID](c)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			return nil
		})
		req, _ := http.NewRequest(http.MethodGet, "/test-inv?limit=abc", nil)
		resp, _ := app.Test(req)
		resp.Body.Close()
	})

	t.Run("with cursor", func(t *testing.T) {
		expected := uuid.Must(uuid.NewV7())
		cursorJSON, _ := json.Marshal(expected)
		cursorStr := base64.URLEncoding.EncodeToString(cursorJSON)

		app.Get("/test-cursor", func(c fiber.Ctx) error {
			pr, err := parsePaginatedRequest[uuid.UUID](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pr.Cursor != expected {
				t.Errorf("want %v, got %v", expected, pr.Cursor)
			}
			return nil
		})
		req, _ := http.NewRequest(http.MethodGet, "/test-cursor?cursor="+cursorStr, nil)
		resp, _ := app.Test(req)
		resp.Body.Close()
	})

	t.Run("int64 cursor", func(t *testing.T) {
		expected := int64(123)
		cursorJSON, _ := json.Marshal(expected)
		cursorStr := base64.URLEncoding.EncodeToString(cursorJSON)

		app.Get("/test-int64", func(c fiber.Ctx) error {
			pr, err := parsePaginatedRequest[int64](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pr.Cursor != expected {
				t.Errorf("want %d, got %d", expected, pr.Cursor)
			}
			return nil
		})
		req, _ := http.NewRequest(http.MethodGet, "/test-int64?cursor="+cursorStr, nil)
		resp, _ := app.Test(req)
		resp.Body.Close()
	})
}
