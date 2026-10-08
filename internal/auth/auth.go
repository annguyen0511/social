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
	// GenerateToken returns a signed token for userID.
	// GenerateToken trả về token đã ký cho userID.
	GenerateToken(userID int64) (string, error)

	// ParseUserID verifies the token and returns the user it belongs to.
	// ParseUserID kiểm tra token và trả về user mà nó thuộc về.
	ParseUserID(token string) (int64, error)
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

func (a *JWTAuthenticator) GenerateToken(userID int64) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    a.issuer,
		Audience:  jwt.ClaimStrings{a.issuer},
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(a.exp)),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (a *JWTAuthenticator) ParseUserID(token string) (int64, error) {
	parsed, err := jwt.ParseWithClaims(
		token,
		&jwt.RegisteredClaims{},
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
		return 0, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return 0, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}
	return userID, nil
}
