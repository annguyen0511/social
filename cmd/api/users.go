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

type createUserRequest struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	UserName  string `json:"username" validate:"required"`
	Password  string `json:"password" validate:"required"`
}

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
	user := model.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		UserName:  req.UserName,
		Password:  req.Password,
	}
	if err := app.store.User.Create(r.Context(), &user); err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := writeJSON(w, http.StatusCreated, user); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

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
