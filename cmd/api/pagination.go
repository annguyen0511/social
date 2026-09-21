package main

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/annguyen0511/social/internal/store"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Pagination is the data every list endpoint returns inside Response.
type Pagination[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page" example:"1"`
	PageSize   int   `json:"page_size" example:"20"`
	TotalItems int64 `json:"total_items" example:"57"`
	TotalPages int   `json:"total_pages" example:"3"`
}

func newPagination[T any](items []T, page store.PaginationQuery, total int64) Pagination[T] {
	if items == nil {
		items = []T{}
	}

	return Pagination[T]{
		Items:      items,
		Page:       page.Page,
		PageSize:   page.PageSize,
		TotalItems: total,
		TotalPages: int((total + int64(page.PageSize) - 1) / int64(page.PageSize)),
	}
}

// readPagination reads ?page= and ?page_size=, falling back to page 1 and
// defaultPageSize when absent. A page past the last one is not an error: it
// returns no items alongside the real totals.
func readPagination(r *http.Request) (store.PaginationQuery, error) {
	query := r.URL.Query()

	page, err := readPositiveInt(query, "page", 1)
	if err != nil {
		return store.PaginationQuery{}, err
	}

	pageSize, err := readPositiveInt(query, "page_size", defaultPageSize)
	if err != nil {
		return store.PaginationQuery{}, err
	}
	if pageSize > maxPageSize {
		return store.PaginationQuery{}, fmt.Errorf("page_size must be at most %d", maxPageSize)
	}

	// Keeps (page-1)*pageSize from overflowing into a negative OFFSET.
	if page > math.MaxInt/pageSize {
		return store.PaginationQuery{}, fmt.Errorf("page is too large")
	}

	return store.PaginationQuery{Page: page, PageSize: pageSize}, nil
}

func readPositiveInt(query url.Values, key string, fallback int) (int, error) {
	raw := query.Get(key)
	if raw == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return n, nil
}
