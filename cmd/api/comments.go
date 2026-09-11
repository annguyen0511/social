package main

import (
	"net/http"
	"strconv"

	"github.com/annguyen0511/social/internal/model"
	"github.com/go-chi/chi/v5"
)

type createCommentRequest struct {
	UserID  int64  `json:"user_id" validate:"required"`
	Content string `json:"content" validate:"required"`
}

func (app *application) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	var req createCommentRequest
	postIdStr := chi.URLParam(r, "postID")
	postId, err := strconv.ParseInt(postIdStr, 10, 64)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	comment := model.Comment{
		PostID:  postId,
		UserID:  req.UserID,
		Content: req.Content,
	}

	if err := app.store.Comment.Create(r.Context(), &comment); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusCreated, comment); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
