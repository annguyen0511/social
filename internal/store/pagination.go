package store

import (
	"context"
	"database/sql"
)

// PaginationQuery is a validated, 1-based page request. List methods take it
// and return the matching slice together with the total row count, so callers
// can report how many pages exist.
type PaginationQuery struct {
	Page     int
	PageSize int
}

func (q PaginationQuery) Limit() int {
	return q.PageSize
}

func (q PaginationQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// paginate runs countQuery for the total, then pageQuery for one page. Both
// receive args; pageQuery additionally receives LIMIT and OFFSET as the two
// placeholders after them, so with one arg it must use $2 and $3.
//
// pageQuery must ORDER BY a unique key. Without one, rows that tie on the sort
// column can come back in a different order on each request, and OFFSET then
// repeats some rows across pages and skips others.
func paginate[T any](
	ctx context.Context,
	db *sql.DB,
	page PaginationQuery,
	countQuery, pageQuery string,
	args []any,
	scan func(*sql.Rows) (T, error),
) ([]T, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var total int64
	if err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	items := []T{}
	// Past the last page there is nothing to fetch, so skip the second query.
	if int64(page.Offset()) >= total {
		return items, total, nil
	}

	pageArgs := append(append([]any{}, args...), page.Limit(), page.Offset())
	rows, err := db.QueryContext(ctx, pageQuery, pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, rows.Err()
}
