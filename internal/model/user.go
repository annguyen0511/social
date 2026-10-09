package model

import "golang.org/x/crypto/bcrypt"

type User struct {
	ID        int64    `json:"id" db:"id"`
	FirstName string   `json:"first_name" db:"first_name"`
	LastName  string   `json:"last_name" db:"last_name"`
	AvatarURL string   `json:"avatar_url" db:"avatar_url"`
	UserName  string   `json:"username" db:"username"`
	Email     string   `json:"email" db:"email"`
	IsActive  bool     `json:"is_active" db:"is_active"`
	Password  Password `json:"-"`
	CreatedAt string   `json:"created_at" db:"created_at"`
	UpdatedAt string   `json:"updated_at" db:"updated_at"`
} //@name UserViewModel

// SearchedUser is a user as a search returns them: the user, plus whether the
// person searching already follows them. Carrying the flag here lets a result
// row draw its follow button straight away; without it the client would have
// to ask about every row it just received.
//
// SearchedUser là một user theo cách tìm kiếm trả về: thông tin user, kèm
// việc người đang tìm đã theo dõi họ hay chưa. Mang sẵn cờ này giúp mỗi dòng
// kết quả vẽ được ngay nút theo dõi; thiếu nó thì client phải hỏi lại từng
// dòng vừa nhận.
type SearchedUser struct {
	User
	IsFollowing bool `json:"is_following"`
} //@name UserSearchViewModel

// Password keeps the hash apart from the plaintext it was derived from. Only
// the hash is ever stored, and it never reaches a client: User.Password
// carries a json:"-" tag.
//
// Password giữ riêng bản băm, tách khỏi mật khẩu thô sinh ra nó. Chỉ bản băm
// được lưu xuống database và nó không bao giờ ra tới client: User.Password
// mang tag json:"-".
type Password struct {
	Hashed []byte `json:"-"`
}

// SetPassword hashes plain with bcrypt. The plaintext is deliberately not
// kept: it is needed only for the moment of hashing, and holding on to it
// would let it surface in a panic trace, a %+v log line or a heap dump.
//
// SetPassword băm mật khẩu thô bằng bcrypt. Bản thô cố tình không được giữ
// lại: nó chỉ cần thiết đúng lúc băm, giữ lại thì có ngày nó lòi ra trong
// panic trace, một dòng log %+v hay một bản dump bộ nhớ.
func (p *Password) SetPassword(plain string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	p.Hashed = hashed
	return nil
}

// ComparePassword reports whether plain matches the stored hash. It returns
// bcrypt.ErrMismatchedHashAndPassword when it does not.
//
// ComparePassword kiểm tra mật khẩu thô có khớp với hash đã lưu hay không.
// Không khớp thì trả về bcrypt.ErrMismatchedHashAndPassword.
func (p *Password) ComparePassword(plain string) error {
	return bcrypt.CompareHashAndPassword(p.Hashed, []byte(plain))
}
