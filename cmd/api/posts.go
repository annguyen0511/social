package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/annguyen0511/social/internal/model"
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
		writeJSONError(w, http.StatusBadRequest, err.Error())
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
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := writeJSON(w, http.StatusCreated, post); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

}

func (app *application) listPostsHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) getPostHandler(w http.ResponseWriter, r *http.Request) {
	postIdStr := chi.URLParam(r, "postID")
	log.Println(postIdStr)
	postId, err := strconv.ParseInt(postIdStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid post ID")
		return
	}

	ctx := r.Context()
	post, err := app.store.Post.GetById(ctx, postId)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := writeJSON(w, http.StatusOK, post); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (app *application) updatePostHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) deletePostHandler(w http.ResponseWriter, r *http.Request) {

}
