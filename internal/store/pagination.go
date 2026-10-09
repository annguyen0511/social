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
//
// Because both queries get the same args, every parameter has to appear in
// both. A page query that needs an extra one — the reader's own id, say, to
// report whether they liked each row — has no place for it here and must use
// paginateWith instead.
//
// Vì cả hai truy vấn nhận cùng một bộ tham số, mọi tham số đều phải xuất hiện
// ở cả hai. Một page query cần thêm tham số riêng — chẳng hạn id của chính
// người đọc, để báo xem họ đã thích từng dòng hay chưa — thì không có chỗ ở
// đây và phải dùng paginateWith.
func paginate[T any](
	ctx context.Context,
	db *sql.DB,
	page PaginationQuery,
	countQuery, pageQuery string,
	args []any,
	scan func(*sql.Rows) (T, error),
) ([]T, int64, error) {
	return paginateWith(ctx, db, page, countQuery, args, pageQuery, args, scan)
}

// paginateWith is paginate for the case where the two queries need different
// parameters.
//
// Postgres rejects a statement handed more parameters than it declares, so
// passing one query's arguments to the other is not a harmless extra — it
// fails the request outright, and only for the query that does not mention
// them.
//
// paginateWith là paginate cho trường hợp hai truy vấn cần bộ tham số khác
// nhau.
//
// Postgres từ chối một câu lệnh được đưa nhiều tham số hơn số nó khai báo, nên
// truyền tham số của truy vấn này sang truy vấn kia không phải là thứ thừa vô
// hại — nó làm hỏng hẳn request, và chỉ hỏng ở đúng truy vấn không nhắc tới
// chúng.
func paginateWith[T any](
	ctx context.Context,
	db *sql.DB,
	page PaginationQuery,
	countQuery string,
	countArgs []any,
	pageQuery string,
	pageArgs []any,
	scan func(*sql.Rows) (T, error),
) ([]T, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var total int64
	if err := db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	items := []T{}
	// Past the last page there is nothing to fetch, so skip the second query.
	if int64(page.Offset()) >= total {
		return items, total, nil
	}

	withLimit := append(append([]any{}, pageArgs...), page.Limit(), page.Offset())
	rows, err := db.QueryContext(ctx, pageQuery, withLimit...)
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
