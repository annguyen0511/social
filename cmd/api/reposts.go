package main

import (
	"errors"
	"net/http"

	"github.com/annguyen0511/social/internal/store"
)

// repostHandler godoc
//
//	@Summary		Repost a post
//	@Description	Shares someone's post under your name. Idempotent: reposting something already reposted still returns the same state. Rejected with 403 while either user blocks the other.
//	@Tags			Post
//	@Produce		json
//	@Param			postID	path		int	true	"Post ID"
//	@Success		200		{object}	RepostViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		401		{object}	JSONError
//	@Failure		403		{object}	JSONError	"A block exists between you and the author"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID}/repost [put]
func (app *application) repostHandler(w http.ResponseWriter, r *http.Request) {
	app.setReposted(w, r, true)
}

// unrepostHandler godoc
//
//	@Summary		Undo a repost
//	@Description	Idempotent: undoing a repost that is not there still returns the same state. Allowed even while a block exists, so a repost can always be taken back.
//	@Tags			Post
//	@Produce		json
//	@Param			postID	path		int	true	"Post ID"
//	@Success		200		{object}	RepostViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		401		{object}	JSONError
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID}/repost [delete]
func (app *application) unrepostHandler(w http.ResponseWriter, r *http.Request) {
	app.setReposted(w, r, false)
}

type repostState struct {
	RepostCount int64 `json:"repost_count" example:"4"`
	IsReposted  bool  `json:"is_reposted" example:"true"`
} //@name RepostViewModel

// setReposted applies the change and answers with the resulting state.
//
// Returning the new count rather than an empty body is what lets the client
// stop guessing. A button that adds one locally is wrong the moment somebody
// else reposts the same post, and it has no way to find out.
//
// setReposted áp dụng thay đổi rồi trả về trạng thái sau đó.
//
// Trả về số đếm mới thay vì thân rỗng chính là thứ giúp client thôi phải
// đoán. Một cái nút tự cộng thêm một ở phía client sẽ sai ngay khi có người
// khác cũng vừa repost bài đó, mà nó thì không có cách nào biết được.
func (app *application) setReposted(w http.ResponseWriter, r *http.Request, reposted bool) {
	// The post comes from postContextMiddileware, which is what turns an
	// unknown id into a 404 before anything is written.
	//
	// Bài viết đến từ postContextMiddileware, và chính nó biến một id không
	// tồn tại thành 404 trước khi có gì được ghi xuống.
	post, ok := getPostFromContext(r)
	if !ok {
		app.internalServerError(w, r, errPostMissing)
		return
	}

	userID := authUser(r).ID
	ctx := r.Context()

	var err error
	if reposted {
		err = app.store.Repost.Repost(ctx, post.ID, userID)
	} else {
		err = app.store.Repost.Unrepost(ctx, post.ID, userID)
	}
	if err != nil {
		switch {
		case errors.Is(err, store.ErrBlocked):
			app.forbiddenResponse(w, r, err)
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	count, isReposted, err := app.store.Repost.Stats(ctx, post.ID, userID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, repostState{RepostCount: count, IsReposted: isReposted}, "repost updated successfully")
}
