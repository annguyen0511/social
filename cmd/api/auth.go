package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
	"github.com/go-chi/chi/v5"
)

type createUserRequest struct {
	FirstName string `json:"first_name" validate:"required,max=100,min=2" example:"An"`
	LastName  string `json:"last_name" validate:"required,max=100,min=2" example:"Nguyen"`
	Email     string `json:"email" validate:"required,email" example:"an.nguyen@example.com"`
	UserName  string `json:"username" validate:"required,max=100,min=2" example:"an.nguyen"`
	Password  string `json:"password" validate:"required,min=8" example:"password123"`
} //@name UserRegisterModel

type registeredUser struct {
	User model.User `json:"user"`
	// Token is the plaintext invitation token. It belongs in an activation
	// email, not in an HTTP response, so it is only filled in outside
	// production, where there is no mailer to deliver it. Drop the field
	// entirely once one exists.
	//
	// Token là chuỗi token gốc. Đúng ra nó phải nằm trong email kích hoạt chứ
	// không phải trong response HTTP, nên chỉ được điền khi chạy ngoài môi
	// trường production, nơi chưa có mailer để gửi đi. Có mailer rồi thì bỏ
	// hẳn trường này.
	Token string `json:"token,omitempty" example:"3f2a...c81d"`
} //@name UserRegisteredViewModel

// newToken returns a 64 character hex string backed by 32 random bytes, for
// both the activation link and the password-reset link. crypto/rand, not
// math/rand: a guessable token would let anyone activate someone else's
// account, or walk into it through the reset form.
//
// newToken trả về chuỗi hex 64 ký tự sinh từ 32 byte ngẫu nhiên, dùng cho cả
// link kích hoạt lẫn link đặt lại mật khẩu. Dùng crypto/rand chứ không phải
// math/rand: token đoán được đồng nghĩa với việc ai cũng kích hoạt được tài
// khoản người khác, hoặc đi thẳng vào nó qua biểu mẫu đặt lại mật khẩu.
func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// registerHandler godoc
//
//	@Summary		Register a user
//	@Description	Creates an inactive account and the invitation that activates it, in one transaction, then emails the activation link. If the mail cannot be sent the registration is rolled back. The password is stored bcrypt-hashed and never returned. The plaintext token is echoed back outside production so the flow can be tested without a mailbox.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		createUserRequest	true	"Account to create"
//	@Success		201		{object}	UserRegisteredViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		409		{object}	JSONError	"The email or username is taken"
//	@Failure		500		{object}	JSONError
//	@Router			/authentication/register [post]
func (app *application) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &model.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		UserName:  req.UserName,
		// The account stays unusable until the invitation is redeemed.
		// Tài khoản chưa dùng được cho tới khi lời mời được kích hoạt.
		IsActive: false,
	}

	if err := user.Password.SetPassword(req.Password); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	token, err := newToken()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.store.User.CreateAndInvited(r.Context(), user, token, app.config.mail.exp); err != nil {
		switch {
		case errors.Is(err, store.ErrDuplicateEmail), errors.Is(err, store.ErrDuplicateUsername):
			app.conflictResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	// The mail goes out after the transaction commits: holding a database
	// transaction open across a network call to SendGrid would pin a
	// connection and its locks for the whole round trip.
	//
	// Mail được gửi sau khi transaction commit: giữ transaction mở suốt một
	// lượt gọi mạng tới SendGrid sẽ ghim connection cùng các khoá của nó
	// trong toàn bộ thời gian chờ.
	activationURL := fmt.Sprintf("%s/confirm/%s", app.config.mail.frontendURL, token)
	if err := app.mailer.SendActivation(user.Email, user.UserName, activationURL); err != nil {
		// Undo the registration. Keeping the account would hold its email
		// and username while no activation link exists to unlock it, and the
		// person could not register again.
		//
		// Huỷ việc đăng ký. Giữ lại tài khoản sẽ chiếm mất email và username
		// trong khi không có đường dẫn kích hoạt nào để mở khoá, và người
		// dùng cũng không đăng ký lại được.
		app.logger.Errorw("activation mail failed, rolling back registration",
			"user_id", user.ID, "email", user.Email, "error", err)

		if delErr := app.store.User.Delete(r.Context(), user.ID); delErr != nil {
			app.logger.Errorw("rollback failed, user left unactivatable",
				"user_id", user.ID, "error", delErr)
		}

		app.internalServerError(w, r, err)
		return
	}

	// Outside production the token also comes back in the response, so the
	// flow can be exercised without opening a mailbox.
	//
	// Ngoài production, token còn được trả lại trong response để chạy thử
	// luồng này mà không cần mở hòm thư.
	body := registeredUser{User: *user}
	if app.config.env != "production" {
		body.Token = token
	}

	app.jsonResponse(w, r, http.StatusCreated, body, "user registered successfully")
}

// activateUserHandler godoc
//
//	@Summary		Activate a user
//	@Description	Redeems an invitation token and marks the account active. The token is consumed, so a second call with the same token returns 404.
//	@Tags			Authentication
//	@Produce		json
//	@Param			token	path		string	true	"Invitation token from registration"
//	@Success		200		{object}	MessageResponse
//	@Failure		404		{object}	JSONError	"The token is unknown, already used, or expired"
//	@Failure		500		{object}	JSONError
//	@Router			/authentication/activate/{token} [put]
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if err := app.store.User.Activate(r.Context(), token); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "user activated successfully")
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"an.nguyen@example.com"`
	Password string `json:"password" validate:"required,min=8" example:"password123"`
} //@name UserLoginModel

// loginHandler godoc
//
//	@Summary		Log in
//	@Description	Verifies the credentials and starts a session. The signed token is returned in an HttpOnly cookie, not in the body, so page scripts cannot read it. An account that has not been activated is refused with 403.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		loginRequest	true	"Credentials"
//	@Success		200		{object}	UserViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		401		{object}	JSONError	"Wrong email or password"
//	@Failure		403		{object}	JSONError	"The account has not been activated"
//	@Failure		500		{object}	JSONError
//	@Router			/authentication/login [post]
func (app *application) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.store.User.GetByEmail(r.Context(), req.Email)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			// Same answer as a wrong password, see unauthorizedResponse.
			// Trả lời giống hệt trường hợp sai mật khẩu, xem unauthorizedResponse.
			app.unauthorizedResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := user.Password.ComparePassword(req.Password); err != nil {
		app.unauthorizedResponse(w, r, err)
		return
	}

	// Checked after the password, so the activation state of an account is
	// only revealed to whoever already knows its password.
	//
	// Kiểm sau khi so mật khẩu, để trạng thái kích hoạt của một tài khoản chỉ
	// lộ ra với người vốn đã biết mật khẩu của nó.
	if !user.IsActive {
		app.forbiddenResponse(w, r, errors.New("account is not activated"))
		return
	}

	if err := app.startSession(w, user); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, user, "logged in successfully")
}

// logoutHandler godoc
//
//	@Summary		Log out
//	@Description	Clears the session cookie. The token itself stays valid until it expires, so this ends the session for this browser only.
//	@Tags			Authentication
//	@Produce		json
//	@Success		200	{object}	MessageResponse
//	@Router			/authentication/logout [post]
func (app *application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	app.clearSessionCookie(w)
	app.jsonResponse(w, r, http.StatusOK, nil, "logged out successfully")
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email" example:"an.nguyen@example.com"`
} //@name ForgotPasswordModel

type forgotPasswordResult struct {
	// Token is the plaintext reset token, filled in only outside production
	// where the mail may be going nowhere. It belongs in an email, never in
	// an HTTP response: anyone who can read this body can take the account.
	//
	// Token là token đặt lại dạng gốc, chỉ được điền khi chạy ngoài
	// production, nơi thư có thể chẳng đi đâu cả. Đúng ra nó phải nằm trong
	// email chứ không bao giờ nằm trong response HTTP: ai đọc được body này
	// là chiếm được tài khoản.
	Token string `json:"token,omitempty" example:"3f2a...c81d"`
} //@name ForgotPasswordViewModel

// forgotPasswordHandler godoc
//
//	@Summary		Ask for a password reset link
//	@Description	Mails a one-time link for choosing a new password. The answer is always 200, whether or not an account exists for the address, so this endpoint cannot be used to find out who is registered. A link is issued at most once every two minutes per account. Outside production the token is echoed back so the flow can be tested without a mailbox.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		forgotPasswordRequest	true	"The address to send the link to"
//	@Success		200		{object}	ForgotPasswordViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/authentication/forgot-password [post]
func (app *application) forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Every path below this line answers the same way. An unknown address, a
	// known one, an account still waiting for activation, a request made too
	// soon — all 200 with nothing to distinguish them. Saying "no such
	// account" here would turn this endpoint into a way of testing whether
	// someone is a member, which is exactly what a forgotten-password form
	// must not be.
	//
	// Mọi nhánh bên dưới dòng này đều trả lời giống hệt nhau. Địa chỉ lạ, địa
	// chỉ có thật, tài khoản còn chờ kích hoạt, hay yêu cầu gửi quá sớm —
	// tất cả đều là 200 và không có gì để phân biệt. Trả lời "không có tài
	// khoản này" sẽ biến endpoint thành công cụ dò xem ai đã đăng ký, đúng
	// thứ mà một biểu mẫu quên-mật-khẩu không được phép là.
	token, err := app.issueResetToken(r, req.Email)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	body := forgotPasswordResult{}
	if app.config.env != "production" {
		body.Token = token
	}

	app.jsonResponse(w, r, http.StatusOK, body, "if that address has an account, a reset link is on its way")
}

// issueResetToken does the work behind the uniform answer. It returns an
// empty token, and no error, for every ordinary reason not to send a mail.
// An error means something actually broke, which is the only case the caller
// should report differently.
//
// issueResetToken làm phần việc nằm sau câu trả lời đồng nhất kia. Nó trả về
// token rỗng, và không có lỗi, với mọi lý do thông thường khiến thư không
// được gửi. Có error nghĩa là thật sự có thứ gì đó hỏng, và đó là trường hợp
// duy nhất phía gọi nên báo khác đi.
func (app *application) issueResetToken(r *http.Request, email string) (string, error) {
	user, err := app.store.User.GetByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	// An account that was never activated has no password worth resetting,
	// and its owner has an activation link to use instead. Sending a reset
	// mail here would also let someone else's unconfirmed registration of
	// your address turn into a working account.
	//
	// Tài khoản chưa từng kích hoạt thì không có mật khẩu nào đáng đặt lại,
	// mà chủ của nó đã có sẵn link kích hoạt để dùng. Gửi thư đặt lại ở đây
	// còn khiến việc người khác đăng ký bằng địa chỉ của bạn mà chưa xác nhận
	// có thể biến thành một tài khoản dùng được.
	if !user.IsActive {
		app.logger.Infow("reset requested for an inactive account", "user_id", user.ID)
		return "", nil
	}

	token, err := newToken()
	if err != nil {
		return "", err
	}

	if err := app.store.User.CreateResetToken(r.Context(), user.ID, token, app.config.mail.exp); err != nil {
		if errors.Is(err, store.ErrTooSoon) {
			app.logger.Infow("reset asked for again within the cooldown", "user_id", user.ID)
			return "", nil
		}
		return "", err
	}

	resetURL := fmt.Sprintf("%s/reset/%s", app.config.mail.frontendURL, token)
	if err := app.mailer.SendPasswordReset(user.Email, user.UserName, resetURL); err != nil {
		// Unlike registration there is nothing to roll back: the account
		// already existed and is untouched. The stored token simply goes
		// unused and expires, and the person can ask again once the cooldown
		// passes.
		//
		// Khác với lúc đăng ký, ở đây không có gì phải hoàn tác: tài khoản
		// vốn đã tồn tại và không hề bị động tới. Token đã lưu chỉ đơn giản
		// là không ai dùng rồi hết hạn, và người dùng xin lại được sau khi
		// hết thời gian chờ.
		app.logger.Errorw("password reset mail failed", "user_id", user.ID, "error", err)
		return token, nil
	}

	return token, nil
}

type resetPasswordRequest struct {
	Token    string `json:"token" validate:"required" example:"3f2a...c81d"`
	Password string `json:"password" validate:"required,min=8" example:"newpassword123"`
} //@name ResetPasswordModel

// resetPasswordHandler godoc
//
//	@Summary		Set a new password with a reset link
//	@Description	Redeems a reset token and writes the new password. The token is consumed, so a second call with the same one returns 404, and every session the account had open is ended — including on other devices. The caller is not signed in afterwards; they log in with the new password.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		resetPasswordRequest	true	"The token from the mail, and the new password"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	JSONError
//	@Failure		404		{object}	JSONError	"The token is unknown, already used, or expired"
//	@Failure		500		{object}	JSONError
//	@Router			/authentication/reset-password [post]
func (app *application) resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var password model.Password
	if err := password.SetPassword(req.Password); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.store.User.ResetPassword(r.Context(), req.Token, password.Hashed); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	// Whoever did this was not signed in, and must not become signed in by
	// doing it: a reset link arriving in a mailbox is weaker proof than a
	// password, so it buys the right to set one, not a session.
	//
	// Người vừa làm việc này không ở trạng thái đăng nhập, và không được trở
	// thành đăng nhập nhờ nó: một link tới hòm thư là bằng chứng yếu hơn mật
	// khẩu, nên nó đổi được quyền đặt mật khẩu mới, chứ không đổi được một
	// phiên.
	app.clearSessionCookie(w)
	app.jsonResponse(w, r, http.StatusOK, nil, "password changed, you can sign in now")
}
