package main

import (
	"net/http"
)

// saveHandler godoc
//
//	@Summary		Save a post
//	@Description	Adds the post to your saved list. Idempotent: saving a post already saved still returns the same state. Saving is private — nobody else is told, and no count is published.
//	@Tags			Post
//	@Produce		json
//	@Param			postID	path		int	true	"Post ID"
//	@Success		200		{object}	SaveViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		401		{object}	JSONError
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID}/save [put]
func (app *application) saveHandler(w http.ResponseWriter, r *http.Request) {
	app.setSaved(w, r, true)
}

// unsaveHandler godoc
//
//	@Summary		Remove a post from your saved list
//	@Description	Idempotent: removing a post that was not saved still returns the same state.
//	@Tags			Post
//	@Produce		json
//	@Param			postID	path		int	true	"Post ID"
//	@Success		200		{object}	SaveViewModelResponse
//	@Failure		400		{object}	JSONError
//	@Failure		401		{object}	JSONError
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/posts/{postID}/save [delete]
func (app *application) unsaveHandler(w http.ResponseWriter, r *http.Request) {
	app.setSaved(w, r, false)
}

type saveState struct {
	IsSaved bool `json:"is_saved" example:"true"`
} //@name SaveViewModel

// setSaved applies the change and answers with the resulting state.
//
// Unlike a like there is no number to report, because a save is private. What
// comes back is only whether this one reader has the post saved, read after
// the write rather than assumed from it.
//
// setSaved áp dụng thay đổi rồi trả về trạng thái sau đó.
//
// Khác với lượt thích, ở đây không có con số nào để báo, vì việc lưu bài là
// riêng tư. Thứ trả về chỉ là người đọc này có đang lưu bài hay không, và nó
// được đọc lại sau khi ghi chứ không phải suy ra từ lệnh ghi.
func (app *application) setSaved(w http.ResponseWriter, r *http.Request, saved bool) {
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
	if saved {
		err = app.store.Save.Save(ctx, post.ID, userID)
	} else {
		err = app.store.Save.Unsave(ctx, post.ID, userID)
	}
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	isSaved, err := app.store.Save.IsSaved(ctx, post.ID, userID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, saveState{IsSaved: isSaved}, "save updated successfully")
}

// listSavedHandler godoc
//
//	@Summary		List the posts I saved
//	@Description	Your own saved posts, most recently saved first. There is no equivalent for another user: a saved list is private, which is why this path says "me" rather than taking an id.
//	@Tags			User
//	@Produce		json
//	@Param			page		query		int	false	"Page number, starting at 1"	default(1)	minimum(1)
//	@Param			page_size	query		int	false	"Items per page"				default(20)	minimum(1)	maximum(100)
//	@Success		200			{object}	FeedPostViewModelPaginationResponse
//	@Failure		400			{object}	JSONError
//	@Failure		401			{object}	JSONError
//	@Failure		500			{object}	JSONError
//	@Router			/users/me/saved [get]
func (app *application) listSavedHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	posts, total, err := app.store.Post.GetSaved(r.Context(), authUser(r).ID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(posts, page, total), "saved posts retrieved successfully")
}
