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
		// One gate for everything hanging off /posts/{postID}: reading it,
		// commenting, liking, saving, reposting. Editing and deleting pass
		// through untouched, because those are owner-only and nobody blocks
		// themselves.
		//
		// Một cánh cổng duy nhất cho mọi thứ nằm dưới /posts/{postID}: đọc
		// bài, bình luận, thích, lưu, repost. Sửa và xoá đi qua không vướng
		// gì, vì hai việc đó chỉ chủ bài làm được mà không ai tự chặn mình.
		if app.hideWhenBlocked(w, r, post.UserID) {
			return
		}

		ctx = context.WithValue(ctx, postContextKey, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getPostFromContext(r *http.Request) (*model.Post, bool) {
	post, ok := r.Context().Value(postContextKey).(*model.Post)
	return post, ok
}

// requireOwnPost returns the post in the route, but only to the person who
// wrote it.
//
// Until now update and delete took the post straight from the context, so any
// signed-in user could edit or delete anyone else's post by knowing its id.
// The check belongs here rather than in each handler, so a future handler that
// changes a post cannot forget it.
//
// requireOwnPost trả về bài viết trên route, nhưng chỉ cho đúng người đã viết
// nó.
//
// Trước đây update và delete lấy thẳng bài từ context, nên bất kỳ ai đã đăng
// nhập cũng sửa hoặc xoá được bài của người khác chỉ cần biết id. Phép kiểm
// đặt ở đây chứ không đặt trong từng handler, để sau này có thêm handler nào
// sửa bài thì cũng không thể quên.
var errPostMissing = errors.New("post missing from request context")

func (app *application) requireOwnPost(w http.ResponseWriter, r *http.Request) (*model.Post, bool) {
	post, ok := getPostFromContext(r)
	if !ok {
		app.internalServerError(w, r, errPostMissing)
		return nil, false
	}

	if post.UserID != authUser(r).ID {
		app.forbiddenResponse(w, r, errors.New("you can only change your own posts"))
		return nil, false
	}

	return post, true
}

type createPostRequest struct {
	Content string   `json:"content" validate:"required,max=1000" example:"Optimistic locking is one extra predicate in the WHERE clause."`
	Title   string   `json:"title" validate:"required,max=100" example:"Optimistic locking in practice"`
	Tags    []string `json:"tags" example:"go,postgres"`
} //@name PostCreateModel

// createPostHandler godoc
//
//	@Summary		Create a post
//	@Description	Creates a post owned by the signed-in user.
//	@Tags			Post
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		createPostRequest	true	"Post to create"
//	@Success		201		{object}	PostViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts [post]
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
		UserID:  authUser(r).ID,
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
	Title   *string   `json:"title" validate:"omitempty,max=100" example:"An updated title"`
	Content *string   `json:"content" validate:"omitempty,max=1000" example:"Updated content."`
	Tags    []*string `json:"tags" validate:"omitempty" example:"go,api"`
} //@name PostUpdateModel

// updatePostHandler godoc
//
//	@Summary		Update a post
//	@Description	Partially updates a post: omitted fields keep their value. Guarded by optimistic locking on the version column, so a concurrent edit is rejected with 409.
//	@Tags			Post
//	@Accept			json
//	@Produce		json
//	@Param			postID	path		int					true	"Post ID"
//	@Param			payload	body		updatePostRequest	true	"Fields to change"
//	@Success		200		{object}	PostViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		403		{object}	JSONError	"The post belongs to someone else"
//	@Failure		404		{object}	JSONError
//	@Failure		409		{object}	JSONError	"The post changed since it was read"
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID} [patch]
func (app *application) updatePostHandler(w http.ResponseWriter, r *http.Request) {
	post, ok := app.requireOwnPost(w, r)
	if !ok {
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

// getPostHandler godoc
//
//	@Summary		Get a post
//	@Description	Returns a post together with its comments, newest first.
//	@Tags			Post
//	@Produce		json
//	@Param			postID	path		int	true	"Post ID"
//	@Success		200		{object}	PostViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID} [get]
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

	// GetById knows nothing about who is asking, so the like state is read
	// here, where the signed-in user is known.
	//
	// GetById không biết gì về người đang hỏi, nên trạng thái thích được đọc ở
	// đây, nơi đã biết ai là người đăng nhập.
	likeCount, isLiked, err := app.store.Like.Stats(r.Context(), post.ID, authUser(r).ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	post.LikeCount = likeCount
	post.IsLiked = isLiked

	isSaved, err := app.store.Save.IsSaved(r.Context(), post.ID, authUser(r).ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	post.IsSaved = isSaved

	repostCount, isReposted, err := app.store.Repost.Stats(r.Context(), post.ID, authUser(r).ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	post.RepostCount = repostCount
	post.IsReposted = isReposted

	err = app.jsonResponse(w, r, http.StatusOK, post, "Success")
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// deletePostHandler godoc
//
//	@Summary		Delete a post
//	@Description	Deletes a post and, through ON DELETE CASCADE, its comments and likes.
//	@Tags			Post
//	@Produce		json
//	@Param			postID	path	int	true	"Post ID"
//	@Success		200		"Post deleted; the response has no body"
//	@Failure		400		{object}	JSONError
//	@Failure		403		{object}	JSONError	"The post belongs to someone else"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID} [delete]
func (app *application) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	post, ok := app.requireOwnPost(w, r)
	if !ok {
		return
	}

	if err := app.store.Post.Delete(r.Context(), post.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
