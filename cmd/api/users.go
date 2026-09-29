package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

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
