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

type contextKey string

const postContextKey contextKey = "post"

func (app *application) postContextMiddileware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		ctx = context.WithValue(ctx, postContextKey, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getPostFromContext(r *http.Request) (*model.Post, bool) {
	post, ok := r.Context().Value(postContextKey).(*model.Post)
	return post, ok
}

type createPostRequest struct {
	Content string   `json:"content" validate:"required,max=1000"`
	Title   string   `json:"title" validate:"required,max=100"`
	Tags    []string `json:"tags"`
}

func (app *application) createPostHandler(w http.ResponseWriter, r *http.Request) {

	var req createPostRequest
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
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
	err := app.store.Post.Create(ctx, &post)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	err = app.jsonResponse(w, r, http.StatusCreated, post, "Post created successfully")
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

}

type updatePostRequest struct {
	Title   *string   `json:"title" validate:"omitempty,max=100"`
	Content *string   `json:"content" validate:"omitempty,max=1000"`
	Tags    []*string `json:"tags" validate:"omitempty"`
}

func (app *application) updatePostHandler(w http.ResponseWriter, r *http.Request) {
	post, ok := getPostFromContext(r)
	if !ok {
		app.badRequestResponse(w, r, errors.New("post not found"))
		return
	}

	var req updatePostRequest
	if err := readJSON(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if req.Title != nil {
		post.Title = *req.Title
	}

	if req.Content != nil {
		post.Content = *req.Content
	}

	if req.Tags != nil {
		newTags := make([]string, len(req.Tags))
		for i, tag := range req.Tags {
			newTags[i] = *tag
		}
		post.Tags = newTags
	}

	if err := app.store.Post.Update(r.Context(), post); err != nil {
		switch {
		case errors.Is(err, store.ErrConflict):
			app.conflictResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	err := app.jsonResponse(w, r, http.StatusOK, post, "Post updated successfully")
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

}

func (app *application) listPostsHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) getPostHandler(w http.ResponseWriter, r *http.Request) {
	post, ok := getPostFromContext(r)
	if !ok {
		app.badRequestResponse(w, r, errors.New("post not found"))
		return
	}

	comments, err := app.store.Comment.GetByPostId(r.Context(), post.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	post.Comments = comments

	err = app.jsonResponse(w, r, http.StatusOK, post, "Success")
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	post, ok := getPostFromContext(r)
	if !ok {
		app.badRequestResponse(w, r, errors.New("post not found"))
		return
	}

	if err := app.store.Post.Delete(r.Context(), post.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
