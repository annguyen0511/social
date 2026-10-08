package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/annguyen0511/social/internal/auth"
	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
)

// sessionCookieName holds the signed JWT. The token lives in an HttpOnly
// cookie rather than localStorage: script injected into the page cannot read
// it, and the browser attaches it to every request on its own.
//
// sessionCookieName chứa JWT đã ký. Token nằm trong cookie HttpOnly thay vì
// localStorage: mã độc chèn vào trang không đọc được nó, và trình duyệt tự
// đính kèm vào mọi request.
const sessionCookieName = "session"

const authUserContextKey contextKey = "auth_user"

// setSessionCookie writes the token for the browser to send back.
// setSessionCookie ghi token để trình duyệt gửi lại ở các request sau.
func (app *application) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(app.config.auth.exp.Seconds()),
		HttpOnly: true,
		// Lax still sends the cookie when the user follows a link into the
		// site, but not on cross-site form posts, which blocks the simplest
		// CSRF. Secure is off outside production so plain http://localhost
		// development keeps working.
		//
		// Lax vẫn gửi cookie khi người dùng bấm link vào trang, nhưng không
		// gửi khi bị site khác submit form sang, nên chặn được dạng CSRF đơn
		// giản nhất. Secure tắt khi chạy ngoài production để dev trên
		// http://localhost vẫn hoạt động.
		SameSite: http.SameSiteLaxMode,
		Secure:   app.config.env == "production",
	})
}

// clearSessionCookie expires the cookie. The value is emptied as well, so a
// browser that ignores MaxAge still has nothing useful to send.
//
// clearSessionCookie làm cookie hết hạn. Giá trị cũng bị xoá rỗng, để trình
// duyệt nào bỏ qua MaxAge thì cũng không còn gì hữu ích để gửi.
func (app *application) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   app.config.env == "production",
	})
}

// requireAuth rejects anyone without a valid session and puts the user it
// belongs to in the request context.
//
// The user is loaded from the database on every request rather than trusted
// from the token's claims. A token stays valid until it expires, so without
// this lookup a deleted account would keep working for days.
//
// requireAuth từ chối ai không có phiên hợp lệ và đặt user tương ứng vào
// context của request.
//
// User được đọc lại từ database ở mỗi request thay vì tin vào claim trong
// token. Token vẫn hợp lệ cho tới khi hết hạn, nên thiếu bước tra cứu này thì
// một tài khoản đã bị xoá vẫn dùng được thêm nhiều ngày.
func (app *application) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			app.unauthorizedResponse(w, r, errors.New("no session cookie"))
			return
		}

		userID, err := app.authenticator.ParseUserID(cookie.Value)
		if err != nil {
			// The cookie is useless; clear it so the browser stops sending a
			// token that will never be accepted again.
			//
			// Cookie này vô dụng; xoá đi để trình duyệt thôi gửi một token sẽ
			// không bao giờ được chấp nhận nữa.
			app.clearSessionCookie(w)
			app.unauthorizedResponse(w, r, err)
			return
		}

		user, err := app.store.User.GetById(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrNotFound):
				app.clearSessionCookie(w)
				app.unauthorizedResponse(w, r, errors.New("account no longer exists"))
			default:
				app.internalServerError(w, r, err)
			}
			return
		}

		if !user.IsActive {
			app.forbiddenResponse(w, r, errors.New("account is not activated"))
			return
		}

		ctx := context.WithValue(r.Context(), authUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authUser returns the signed-in user. It is only safe to call behind
// requireAuth, which is why it panics rather than returning a zero user:
// a handler reachable without a session is a routing bug, and silently acting
// as nobody would be worse than crashing one request.
//
// authUser trả về user đang đăng nhập. Chỉ an toàn khi gọi phía sau
// requireAuth, nên nó panic thay vì trả về user rỗng: một handler vào được mà
// không cần phiên là lỗi khai báo route, và âm thầm hành động dưới danh nghĩa
// "không ai cả" còn tệ hơn là để một request chết.
func authUser(r *http.Request) *model.User {
	user, ok := r.Context().Value(authUserContextKey).(*model.User)
	if !ok {
		panic("authUser called on a route that is not behind requireAuth")
	}
	return user
}

var _ auth.Authenticator = (*auth.JWTAuthenticator)(nil)
