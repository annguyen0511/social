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

// newInvitationToken returns a 64 character hex string backed by 32 random
// bytes. crypto/rand, not math/rand: a guessable token would let anyone
// activate someone else's account.
//
// newInvitationToken trả về chuỗi hex 64 ký tự sinh từ 32 byte ngẫu nhiên.
// Dùng crypto/rand chứ không phải math/rand: token đoán được đồng nghĩa với
// việc ai cũng kích hoạt được tài khoản của người khác.
func newInvitationToken() (string, error) {
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

	token, err := newInvitationToken()
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

	token, err := app.authenticator.GenerateToken(user.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.setSessionCookie(w, token)
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
