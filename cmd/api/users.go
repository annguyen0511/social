package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
	"github.com/go-chi/chi/v5"
)

const userContextKey contextKey = "user"

func (app *application) userContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIdStr := chi.URLParam(r, "userID")
		userId, err := strconv.ParseInt(userIdStr, 10, 64)

		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		ctx := r.Context()
		user, err := app.store.User.GetById(ctx, userId)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrNotFound):
				app.notFoundResponse(w, r, err)
				return
			default:
				app.internalServerError(w, r, err)
				return
			}
		}
		ctx = context.WithValue(ctx, userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserFromContext(r *http.Request) (*model.User, bool) {
	user, ok := r.Context().Value(userContextKey).(*model.User)
	return user, ok
}

// getUserHandler godoc
//
//	@Summary	Get a user
//	@Tags		User
//	@Produce	json
//	@Param		userID	path		int	true	"User ID"
//	@Success	200		{object}	UserViewModelResponse
//	@Failure	400		{object}	JSONError
//	@Failure	404		{object}	JSONError
//	@Failure	500		{object}	JSONError
//	@Router		/users/{userID} [get]
func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		app.badRequestResponse(w, r, errors.New("user not found"))
		return
	}

	err := app.jsonResponse(w, r, http.StatusOK, user, "user retrieved successfully")
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// getCurrentUserHandler godoc
//
//	@Summary		Get the signed-in user
//	@Description	Returns the account behind the session cookie. A client has no other way to learn its own ID, which it needs to tell its own profile apart from someone else's.
//	@Tags			User
//	@Produce		json
//	@Success		200	{object}	UserViewModelResponse
//	@Failure		401	{object}	JSONError
//	@Failure		500	{object}	JSONError
//	@Router			/users/me [get]
func (app *application) getCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	if err := app.jsonResponse(w, r, http.StatusOK, authUser(r), "current user retrieved successfully"); err != nil {
		app.internalServerError(w, r, err)
	}
}

type updateProfileRequest struct {
	FirstName *string `json:"first_name" example:"An"`
	LastName  *string `json:"last_name" example:"Nguyen"`
	UserName  *string `json:"username" example:"an.nguyen"`
} //@name UserUpdateModel

// validateProfile checks the user after the request has been applied to it,
// rather than checking the request itself.
//
// The reason is that every field is a pointer so an omitted field can keep its
// old value, and `omitempty` in a validate tag cannot tell "not sent" from
// "sent as an empty string": it skips both, which would let a caller erase a
// name to "". Checking the merged result has no such blind spot.
//
// Lengths count runes, not bytes. Postgres counts characters in varchar(255),
// while len() on a Go string counts bytes, so a Vietnamese name would be
// rejected at roughly a third of the length actually allowed.
//
// The avatar is not checked here: it is never set from a request body, only by
// the upload endpoint, which writes a path this server chose itself.
//
// validateProfile kiểm tra user sau khi request đã được áp vào, chứ không
// kiểm bản thân request.
//
// Lý do: mọi field đều là con trỏ để field không gửi thì giữ giá trị cũ, mà
// `omitempty` trong validate tag không phân biệt được "không gửi" với "gửi
// chuỗi rỗng" — nó bỏ qua cả hai, nên người gọi có thể xoá trắng tên thành
// "". Kiểm kết quả đã hợp nhất thì không còn điểm mù đó.
//
// Độ dài đếm theo rune chứ không phải byte. Postgres đếm ký tự trong
// varchar(255), còn len() của Go đếm byte, nên một cái tên tiếng Việt sẽ bị
// chặn ở khoảng một phần ba độ dài thực sự được phép.
//
// Avatar không được kiểm ở đây: nó không bao giờ đặt từ body của request mà
// chỉ do endpoint tải lên ghi, với đường dẫn do chính server này chọn.
func validateProfile(user *model.User) error {
	for _, field := range []struct{ name, value string }{
		{"first_name", user.FirstName},
		{"last_name", user.LastName},
		{"username", user.UserName},
	} {
		if n := len([]rune(field.value)); n < 2 || n > 100 {
			return fmt.Errorf("%s must be between 2 and 100 characters", field.name)
		}
	}

	return nil
}

// updateProfileHandler godoc
//
//	@Summary		Update my profile
//	@Description	Changes the signed-in user's own profile. Every field is optional: one left out of the body keeps its current value. The avatar has its own endpoints because it is a file, and the email cannot be changed here because it is the login identifier.
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		updateProfileRequest	true	"Fields to change"
//	@Success		200		{object}	UserViewModelResponse
//	@Failure		400		{object}	JSONError	"A name is outside 2-100 characters"
//	@Failure		401		{object}	JSONError
//	@Failure		404		{object}	JSONError	"The account no longer exists"
//	@Failure		409		{object}	JSONError	"The username is taken"
//	@Failure		500		{object}	JSONError
//	@Router			/users/me [patch]
func (app *application) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	var req updateProfileRequest
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Start from the copy requireAuth already loaded for this request, so any
	// field the caller left out keeps the value it has in the database.
	//
	// Bắt đầu từ bản mà requireAuth đã đọc sẵn cho request này, để field nào
	// người gọi không gửi thì giữ nguyên giá trị đang có trong database.
	user := *authUser(r)

	if req.FirstName != nil {
		user.FirstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		user.LastName = strings.TrimSpace(*req.LastName)
	}
	if req.UserName != nil {
		user.UserName = strings.TrimSpace(*req.UserName)
	}

	if err := validateProfile(&user); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.store.User.Update(r.Context(), &user); err != nil {
		switch {
		case errors.Is(err, store.ErrDuplicateUsername):
			app.conflictResponse(w, r, err)
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.jsonResponse(w, r, http.StatusOK, user, "profile updated successfully"); err != nil {
		app.internalServerError(w, r, err)
	}
}

// listUserPostsHandler godoc
//
//	@Summary		List a user's posts
//	@Description	Posts written by the user in the route, newest first. Ties on created_at are broken by post ID, so pages never overlap.
//	@Tags			User
//	@Produce		json
//	@Param			userID		path		int	true	"User ID"
//	@Param			page		query		int	false	"Page number, starting at 1"	default(1)	minimum(1)
//	@Param			page_size	query		int	false	"Items per page"				default(20)	minimum(1)	maximum(100)
//	@Success		200			{object}	FeedPostViewModelPaginationResponse
//	@Failure		400			{object}	JSONError
//	@Failure		401			{object}	JSONError
//	@Failure		404			{object}	JSONError
//	@Failure		500			{object}	JSONError
//	@Router			/users/{userID}/posts [get]
func (app *application) listUserPostsHandler(w http.ResponseWriter, r *http.Request) {
	// userContextMiddleware already loaded the user, which is what turns an
	// unknown id into a 404 instead of an empty list.
	//
	// userContextMiddleware đã đọc sẵn user, và chính điều đó biến một id
	// không tồn tại thành 404 thay vì một danh sách rỗng.
	author, ok := getUserFromContext(r)
	if !ok {
		app.internalServerError(w, r, errors.New("user missing from request context"))
		return
	}

	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	posts, total, err := app.store.Post.GetByUser(r.Context(), author.ID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(posts, page, total), "posts retrieved successfully")
}

// minSearchQuery keeps a one-character query from matching almost everyone.
// Trigram matching needs three characters to use the index well, but two is a
// reasonable floor for short usernames.
//
// minSearchQuery chặn truy vấn một ký tự khớp gần như mọi người. Trigram cần
// ba ký tự mới dùng index hiệu quả, nhưng hai là mức sàn hợp lý cho những
// username ngắn.
const minSearchQuery = 2

// searchUsersHandler godoc
//
//	@Summary		Search users
//	@Description	Finds active users whose username or full name contains the query, ranked by how closely the username matches. The caller is left out, and so is anyone on either side of a block.
//	@Tags			User
//	@Produce		json
//	@Param			q			query		string	true	"Text to search for, at least 2 characters"
//	@Param			page		query		int		false	"Page number, starting at 1"	default(1)	minimum(1)
//	@Param			page_size	query		int		false	"Items per page"				default(20)	minimum(1)	maximum(100)
//	@Success		200			{object}	UserSearchViewModelPaginationResponse
//	@Failure		400			{object}	JSONError	"q is shorter than 2 characters, or the paging parameters are invalid"
//	@Failure		401			{object}	JSONError
//	@Failure		500			{object}	JSONError
//	@Router			/users/search [get]
func (app *application) searchUsersHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < minSearchQuery {
		app.badRequestResponse(w, r, fmt.Errorf("q must be at least %d characters", minSearchQuery))
		return
	}

	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	users, total, err := app.store.User.Search(r.Context(), authUser(r).ID, q, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(users, page, total), "users retrieved successfully")
}
