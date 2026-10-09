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
