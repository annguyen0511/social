package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/annguyen0511/social/internal/model"
	"github.com/lib/pq"
)

type UserStore struct {
	db *sql.DB
}

// Create stores a user on its own. Registration goes through CreateAndInvited
// instead, so that the user and the invitation share one transaction.
//
// Create chỉ lưu mỗi user. Luồng đăng ký dùng CreateAndInvited thay cho hàm
// này, để user và lời mời nằm chung một transaction.
func (u *UserStore) Create(ctx context.Context, user *model.User) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return withTx(ctx, u.db, func(tx *sql.Tx) error {
		return u.create(ctx, tx, user)
	})
}

// CreateAndInvited registers a user and records the invitation that activates
// them. Both statements share a transaction: an account nobody can ever
// activate, or an invitation pointing at a user that was never stored, would
// both be worse than the registration failing outright.
//
// token is the plaintext handed to the user. Only its SHA-256 hash is stored,
// so a leaked database cannot be used to activate anyone's account.
//
// CreateAndInvited đăng ký user và ghi lời mời dùng để kích hoạt tài khoản.
// Hai lệnh nằm chung một transaction: một tài khoản không ai kích hoạt được,
// hoặc một lời mời trỏ tới user chưa từng được tạo, đều tệ hơn là để việc
// đăng ký thất bại hẳn.
//
// token là chuỗi gốc đưa cho người dùng. DB chỉ lưu bản băm SHA-256 của nó,
// nên kẻ lấy được database cũng không kích hoạt hộ tài khoản người khác.
func (u *UserStore) CreateAndInvited(ctx context.Context, user *model.User, token string, exp time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return withTx(ctx, u.db, func(tx *sql.Tx) error {
		if err := u.create(ctx, tx, user); err != nil {
			return err
		}
		return u.createInvitation(ctx, tx, token, user.ID, exp)
	})
}

// Activate turns the invitation in token into an active account. The
// invitation is deleted in the same transaction, so a token works exactly
// once. An unknown or expired token is reported as ErrNotFound.
//
// Activate đổi lời mời trong token thành một tài khoản đã kích hoạt. Lời mời
// bị xoá ngay trong cùng transaction, nên mỗi token chỉ dùng được đúng một
// lần. Token lạ hoặc hết hạn đều được báo là ErrNotFound.
func (u *UserStore) Activate(ctx context.Context, token string) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return withTx(ctx, u.db, func(tx *sql.Tx) error {
		userID, err := u.userIDFromInvitation(ctx, tx, token)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `UPDATE users SET is_active = true, updated_at = NOW() WHERE id = $1`, userID); err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM user_invitations WHERE user_id = $1`, userID)
		return err
	})
}

// create inserts one user. It takes a *sql.Tx rather than the pool so both
// Create and CreateAndInvited can share it, each inside its own transaction.
//
// create chèn một user. Nó nhận *sql.Tx thay vì pool để cả Create lẫn
// CreateAndInvited dùng chung được, mỗi bên trong transaction của mình.
// Delete removes a user. Their invitations go with them through the
// ON DELETE CASCADE on user_invitations.fk_user_invitations_user_id, so the
// database guarantees no invitation can outlive the account it points at,
// whatever deletes the row.
//
// Registration uses this to undo itself when the activation mail cannot be
// sent: leaving the account behind would hold the email and username hostage
// with no way to activate them.
//
// Delete xoá một user. Các lời mời của họ bị xoá theo nhờ ON DELETE CASCADE
// trên khoá ngoại user_invitations.fk_user_invitations_user_id, nên database
// bảo đảm không lời mời nào sống lâu hơn tài khoản nó trỏ tới, bất kể ai xoá.
//
// Luồng đăng ký dùng nó để tự huỷ khi không gửi được mail kích hoạt: để lại
// tài khoản sẽ chiếm mất email và username mà không có cách nào kích hoạt.
func (u *UserStore) Delete(ctx context.Context, userID int64) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := u.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}

func (u *UserStore) create(ctx context.Context, tx *sql.Tx, user *model.User) error {
	query := `
	INSERT INTO users (first_name, last_name, avatar_url, username, email, password, is_active)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id, created_at, updated_at
	`

	err := tx.QueryRowContext(
		ctx,
		query,
		user.FirstName,
		user.LastName,
		user.AvatarURL,
		user.UserName,
		user.Email,
		user.Password.Hashed,
		user.IsActive,
	).
		Scan(
			&user.ID,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

	if err != nil {
		// A taken email or username is an ordinary outcome of registration,
		// not a server fault, so it gets its own error rather than a raw
		// driver error the handler would turn into a 500.
		//
		// Email hoặc username đã có người dùng là kết quả bình thường của
		// việc đăng ký, không phải lỗi server, nên nó có error riêng thay vì
		// lỗi thô của driver mà handler sẽ biến thành 500.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			switch pqErr.Constraint {
			case "users_email_key":
				return ErrDuplicateEmail
			case "users_username_key":
				return ErrDuplicateUsername
			}
		}
		return err
	}
	return nil
}

// createInvitation stores the SHA-256 hash of token, never token itself,
// together with the moment it stops being valid.
//
// createInvitation lưu bản băm SHA-256 của token chứ không lưu token, kèm
// thời điểm token hết hiệu lực.
func (u *UserStore) createInvitation(ctx context.Context, tx *sql.Tx, token string, userID int64, exp time.Duration) error {
	// Clear this user's earlier invitations, and sweep up anyone's expired
	// ones while holding the transaction, so dead rows cannot pile up
	// without a scheduled job. Only the newest invitation stays usable.
	//
	// Xoá các lời mời cũ của chính user này, đồng thời quét luôn lời mời hết
	// hạn của mọi user ngay trong transaction, để dòng chết không tích tụ mà
	// không cần job định kỳ. Chỉ lời mời mới nhất còn dùng được.
	cleanup := `DELETE FROM user_invitations WHERE user_id = $1 OR expired_at <= NOW()`
	if _, err := tx.ExecContext(ctx, cleanup, userID); err != nil {
		return err
	}

	query := `
	INSERT INTO user_invitations (token, user_id, expired_at)
	VALUES ($1, $2, $3)
	`

	hash := sha256.Sum256([]byte(token))
	_, err := tx.ExecContext(ctx, query, hash[:], userID, time.Now().Add(exp))
	return err
}

// userIDFromInvitation looks the token up by its hash and rejects expired
// rows in the same WHERE clause, so an expired token is indistinguishable
// from an unknown one to the caller.
//
// userIDFromInvitation tra token theo bản băm và loại luôn dòng hết hạn ngay
// trong mệnh đề WHERE, nên với người gọi thì token hết hạn và token không tồn
// tại là như nhau.
func (u *UserStore) userIDFromInvitation(ctx context.Context, tx *sql.Tx, token string) (int64, error) {
	query := `
	SELECT user_id
	FROM user_invitations
	WHERE token = $1 AND expired_at > NOW()
	`

	hash := sha256.Sum256([]byte(token))

	var userID int64
	if err := tx.QueryRowContext(ctx, query, hash[:]).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return userID, nil
}

// GetByEmail loads the account that logs in with this address, hash included.
// Email is a citext column so the lookup is case insensitive, which matches
// what people expect when typing their address.
//
// GetByEmail lấy tài khoản đăng nhập bằng địa chỉ này, kèm cả hash mật khẩu.
// Cột email có kiểu citext nên tra cứu không phân biệt hoa thường, đúng như
// người dùng mong đợi khi gõ địa chỉ của mình.
func (u *UserStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `
	SELECT id, first_name, last_name, COALESCE(avatar_url, ''), username, email, password, is_active, created_at, updated_at
	FROM users
	WHERE email = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := u.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.UserName,
		&user.Email,
		&user.Password.Hashed,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

// Search finds active users whose username or full name contains q, ranked by
// how closely the username matches. viewerID is excluded, and so is anyone on
// either side of a block: the point of blocking is to disappear from each
// other, and search would be the easiest way around that.
//
// ILIKE with a leading wildcard normally forces a full scan, but the GIN
// trigram indexes make it an index lookup. The expressions here must stay
// character for character identical to the ones the indexes are built on.
//
// Search tìm những user đang hoạt động có username hoặc họ tên chứa q, xếp
// theo mức khớp của username. Loại chính người tìm, và loại cả hai chiều của
// quan hệ chặn: mục đích của block là biến mất khỏi nhau, mà tìm kiếm lại là
// đường vòng dễ nhất.
//
// ILIKE mở đầu bằng ký tự đại diện vốn buộc quét toàn bảng, nhưng index GIN
// trigram biến nó thành tra cứu theo index. Các biểu thức ở đây phải trùng
// từng ký tự với biểu thức dùng để tạo index.
func (u *UserStore) Search(ctx context.Context, viewerID int64, q string, page PaginationQuery) ([]model.SearchedUser, int64, error) {
	filter := `
	FROM users u
	WHERE u.is_active
	  AND u.id <> $1
	  AND (u.username ILIKE '%' || $2 || '%'
	       OR (u.first_name || ' ' || u.last_name) ILIKE '%' || $2 || '%')
	  AND NOT EXISTS (
	        SELECT 1 FROM blocks b
	        WHERE (b.blocker_id = $1 AND b.blocked_id = u.id)
	           OR (b.blocker_id = u.id AND b.blocked_id = $1)
	      )
	`

	countQuery := `SELECT COUNT(*) ` + filter

	// The EXISTS subquery runs only for the rows on this page, since it sits
	// in the SELECT list rather than the WHERE clause, and it hits the primary
	// key of follows. That is what keeps one query cheaper than the client
	// asking about every row it just received.
	//
	// Truy vấn con EXISTS chỉ chạy cho những dòng thuộc trang này, vì nó nằm
	// trong danh sách SELECT chứ không phải WHERE, và nó đi theo khoá chính
	// của bảng follows. Nhờ vậy một câu truy vấn vẫn rẻ hơn việc client hỏi
	// lại từng dòng vừa nhận.
	pageQuery := `
	SELECT u.id, u.first_name, u.last_name, COALESCE(u.avatar_url, ''), u.username,
	       u.email, u.is_active, u.created_at, u.updated_at,
	       EXISTS (
	         SELECT 1 FROM follows f
	         WHERE f.follower_id = $1 AND f.following_id = u.id
	       ) AS is_following
	` + filter + `
	ORDER BY similarity(u.username, $2) DESC, u.id DESC
	LIMIT $3 OFFSET $4
	`

	return paginate(ctx, u.db, page, countQuery, pageQuery, []any{viewerID, q}, func(rows *sql.Rows) (model.SearchedUser, error) {
		var user model.SearchedUser
		err := rows.Scan(
			&user.ID, &user.FirstName, &user.LastName, &user.AvatarURL, &user.UserName,
			&user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
			&user.IsFollowing,
		)
		return user, err
	})
}

// Update writes the fields a user is allowed to change about themselves.
// Email is deliberately not among them: it is the login identifier, so
// changing it without proving ownership of the new address would let someone
// lock themselves out, or claim an address they do not hold.
//
// Update ghi những field mà người dùng được phép tự đổi. Email cố tình không
// nằm trong số đó: nó là định danh để đăng nhập, nên đổi mà không chứng minh
// được quyền sở hữu địa chỉ mới sẽ khiến người ta tự khoá mình ra ngoài, hoặc
// chiếm một địa chỉ không phải của mình.
func (u *UserStore) Update(ctx context.Context, user *model.User) error {
	query := `
	UPDATE users
	SET first_name = $1, last_name = $2, avatar_url = $3, username = $4, updated_at = NOW()
	WHERE id = $5 RETURNING updated_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := u.db.QueryRowContext(
		ctx, query,
		user.FirstName, user.LastName, user.AvatarURL, user.UserName, user.ID,
	).Scan(&user.UpdatedAt)

	if err != nil {
		// No row means the account was deleted between the session check and
		// this write. That is a 404, not a server fault.
		//
		// Không có dòng nào nghĩa là tài khoản đã bị xoá trong khoảng giữa lúc
		// kiểm phiên và lúc ghi. Đó là 404, không phải lỗi server.
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}

		// Someone else already holds the requested username. Ordinary outcome,
		// so it gets its own error rather than a driver error the handler
		// would turn into a 500.
		//
		// Username được yêu cầu đã có người khác dùng. Đây là kết quả bình
		// thường nên nó có error riêng, thay vì lỗi thô của driver mà handler
		// sẽ biến thành 500.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "users_username_key" {
			return ErrDuplicateUsername
		}
		return err
	}
	return nil
}

func (u *UserStore) GetById(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	query := `
	SELECT id, first_name, last_name, COALESCE(avatar_url, ''), username, email, password, is_active, created_at, updated_at
	FROM users
	WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := u.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.UserName,
		&user.Email,
		&user.Password.Hashed,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}
