package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
	"github.com/go-chi/chi/v5"
)

type createPostRequest struct {
	Content string   `json:"content"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
}

func (app *application) createPostHandler(w http.ResponseWriter, r *http.Request) {

	var req createPostRequest
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	post := model.Post{
		Content: req.Content,
		Title:   req.Title,
		Tags:    req.Tags,
		// change after auth
		UserID: 1,
	}

	ctx := r.Context()
	if err := app.store.Post.Create(ctx, &post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusCreated, post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

}

func (app *application) listPostsHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) getPostHandler(w http.ResponseWriter, r *http.Request) {
	postIdStr := chi.URLParam(r, "postID")
	postId, err := strconv.ParseInt(postIdStr, 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()
	post, err := app.store.Post.GetById(ctx, postId)
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

	if err := writeJSON(w, http.StatusOK, post); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) updatePostHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) deletePostHandler(w http.ResponseWriter, r *http.Request) {

}
