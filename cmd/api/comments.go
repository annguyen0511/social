package main

import (
	"net/http"
	"strconv"

	"github.com/annguyen0511/social/internal/model"
	"github.com/go-chi/chi/v5"
)

type createCommentRequest struct {
	UserID  int64  `json:"user_id" validate:"required" example:"2"`
	Content string `json:"content" validate:"required" example:"Clean write-up, easy to follow."`
} //@name CommentCreateModel

// createCommentHandler godoc
//
//	@Summary		Comment on a post
//	@Description	Adds a comment to a post. Unlike most endpoints this one returns the comment itself, not the Response envelope.
//	@Tags			Comment
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		int						true	"Post ID"
//	@Param			payload	body		createCommentRequest	true	"Comment to create"
//	@Success		201		{object}	model.Comment
//	@Failure		400		{object}	JSONError
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID}/comment [post]
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
