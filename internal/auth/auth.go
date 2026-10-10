package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned for anything the caller should treat the same
// way: a forged signature, an expired token, a wrong issuer, a malformed
// string. Telling them apart would only help an attacker probe the format.
//
// ErrInvalidToken trả về cho mọi trường hợp mà phía gọi nên xử lý giống nhau:
// chữ ký giả, token hết hạn, sai issuer, chuỗi hỏng. Phân biệt rạch ròi chỉ
// giúp kẻ tấn công dò ra định dạng.
var ErrInvalidToken = errors.New("invalid authentication token")

// Authenticator issues and verifies the tokens that identify a user. The
// interface keeps the HTTP layer from depending on a specific JWT library, and
// lets tests substitute a fake.
//
// Authenticator phát hành và kiểm tra token định danh người dùng. Interface
// giữ cho tầng HTTP không phụ thuộc vào một thư viện JWT cụ thể, và cho phép
// test thay bằng bản giả.
type Authenticator interface {
	// GenerateToken returns a signed token for userID, stamped with the
	// account's current token version.
	//
	// GenerateToken trả về token đã ký cho userID, đóng dấu kèm phiên bản
	// token hiện tại của tài khoản.
	GenerateToken(userID int64, version int) (string, error)

	// ParseIdentity verifies the token and returns who it belongs to along
	// with the version it was signed under. The caller compares that version
	// against the account's current one; a token alone proves nothing about
	// whether the password has changed since.
	//
	// ParseIdentity kiểm tra token rồi trả về chủ nhân của nó cùng phiên bản
	// lúc ký. Phía gọi phải đối chiếu phiên bản đó với phiên bản hiện tại của
	// tài khoản; bản thân token không nói lên được mật khẩu đã đổi hay chưa.
	ParseIdentity(token string) (userID int64, version int, err error)
}

// claims carries the version alongside the registered fields. It is a
// separate type because jwt.RegisteredClaims has no room for a custom one.
//
// claims mang thêm phiên bản bên cạnh các field chuẩn. Phải là kiểu riêng vì
// jwt.RegisteredClaims không có chỗ cho field tự định nghĩa.
type claims struct {
	jwt.RegisteredClaims
	Version int `json:"ver"`
}

// JWTAuthenticator signs tokens with HMAC-SHA256.
// JWTAuthenticator ký token bằng HMAC-SHA256.
type JWTAuthenticator struct {
	secret []byte
	issuer string
	exp    time.Duration
}

func NewJWT(secret, issuer string, exp time.Duration) *JWTAuthenticator {
	return &JWTAuthenticator{secret: []byte(secret), issuer: issuer, exp: exp}
}

func (a *JWTAuthenticator) GenerateToken(userID int64, version int) (string, error) {
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			Issuer:    a.issuer,
			Audience:  jwt.ClaimStrings{a.issuer},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.exp)),
		},
		Version: version,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (a *JWTAuthenticator) ParseIdentity(token string) (int64, int, error) {
	parsed, err := jwt.ParseWithClaims(
		token,
		&claims{},
		func(t *jwt.Token) (any, error) {
			// Pinning the algorithm is what stops the classic attack of
			// re-signing a token with "alg":"none", or with HMAC against a
			// public RSA key.
			//
			// Ghim cứng thuật toán chính là thứ chặn kiểu tấn công kinh điển:
			// ký lại token với "alg":"none", hoặc dùng HMAC với khoá RSA công
			// khai.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
			}
			return a.secret, nil
		},
		jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return 0, 0, ErrInvalidToken
	}

	c, ok := parsed.Claims.(*claims)
	if !ok {
		return 0, 0, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(c.Subject, 10, 64)
	if err != nil {
		return 0, 0, ErrInvalidToken
	}

	// A token signed before this field existed has no "ver" and decodes to
	// zero, which is exactly the default the migration gave every account.
	// Sessions open at deploy time therefore keep working.
	//
	// Token ký từ trước khi có field này thì không có "ver" và giải mã ra 0,
	// đúng bằng giá trị mặc định mà migration đặt cho mọi tài khoản. Nhờ vậy
	// các phiên đang mở lúc triển khai vẫn chạy tiếp.
	return userID, c.Version, nil
}
