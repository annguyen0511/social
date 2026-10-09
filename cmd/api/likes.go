package main

import (
	"net/http"
)

// likeHandler godoc
//
//	@Summary		Like a post
//	@Description	Idempotent: liking a post already liked still returns the same state. Returns the post's like count and whether you are among them.
//	@Tags			Post
//	@Produce		json
//	@Param			postID	path		int	true	"Post ID"
//	@Success		200		{object}	LikeViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		401		{object}	JSONError
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID}/like [put]
func (app *application) likeHandler(w http.ResponseWriter, r *http.Request) {
	app.setLike(w, r, true)
}

// unlikeHandler godoc
//
//	@Summary		Remove a like
//	@Description	Idempotent: removing a like that is not there still returns the same state.
//	@Tags			Post
//	@Produce		json
//	@Param			postID	path		int	true	"Post ID"
//	@Success		200		{object}	LikeViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		401		{object}	JSONError
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID}/like [delete]
func (app *application) unlikeHandler(w http.ResponseWriter, r *http.Request) {
	app.setLike(w, r, false)
}

type likeState struct {
	LikeCount int64 `json:"like_count" example:"12"`
	IsLiked   bool  `json:"is_liked" example:"true"`
} //@name LikeViewModel

// setLike applies the change and answers with the resulting state.
//
// Returning the new count rather than an empty body is what lets the client
// stop guessing. A button that adds one locally is wrong the moment somebody
// else likes the same post, and it has no way to find out.
//
// setLike áp dụng thay đổi rồi trả về trạng thái sau đó.
//
// Trả về số đếm mới thay vì thân rỗng chính là thứ giúp client thôi phải đoán.
// Một cái nút tự cộng thêm một ở phía client sẽ sai ngay khi có người khác
// cũng thích bài đó, mà nó thì không có cách nào biết được.
func (app *application) setLike(w http.ResponseWriter, r *http.Request, liked bool) {
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
	if liked {
		err = app.store.Like.Like(ctx, post.ID, userID)
	} else {
		err = app.store.Like.Unlike(ctx, post.ID, userID)
	}
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	count, isLiked, err := app.store.Like.Stats(ctx, post.ID, userID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, likeState{LikeCount: count, IsLiked: isLiked}, "like updated successfully")
}
