package handlers

import (
	"encoding/base64"
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

const (
	minPageLimit = 15
	maxPageLimit = 100
)

type PaginatedRequest[T any] struct {
	Cursor T
	Limit  int
}

type PaginatedResponse[T any] struct {
	Items   []T    `json:"items"`
	Cursor  string `json:"cursor,omitempty"`
	HasMore bool   `json:"hasMore"`
}

func parsePaginatedRequest[T any](c fiber.Ctx) (PaginatedRequest[T], error) {
	var pr PaginatedRequest[T]
	cursorStr := c.Query("cursor")

	limitStr := c.Query("limit")
	if limitStr == "" {
		pr.Limit = minPageLimit
	} else {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return pr, errInvalidLimit
		}
		pr.Limit = limit
	}

	if pr.Limit < minPageLimit || pr.Limit > maxPageLimit {
		return pr, errInvalidLimit
	}

	if cursorStr != "" {
		var err error
		pr.Cursor, err = decodeCursor[T](cursorStr)
		if err != nil {
			return pr, err
		}
	}

	return pr, nil
}

func newPaginatedResponse[T any](items []T, limit int) PaginatedResponse[T] {
	return PaginatedResponse[T]{
		Items:   items,
		HasMore: len(items) == limit,
	}
}

func encodeCursor(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(data)
}

func decodeCursor[T any](cursor string) (T, error) {
	var v T
	if cursor == "" {
		return v, nil
	}

	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return v, errInvalidCursor
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return v, errInvalidCursor
	}

	return v, nil
}
