package main

import (
	"net/http"
)

// getFeedHandler godoc
//
//	@Summary		Get my feed
//	@Description	My own posts plus posts by everyone I follow, newest first. Ties on created_at are broken by post ID, so pages never overlap.
//	@Tags			Feed
//	@Produce		json
//	@Param			page		query		int	false	"Page number, starting at 1"	default(1)	minimum(1)
//	@Param			page_size	query		int	false	"Items per page"				default(20)	minimum(1)	maximum(100)
//	@Success		200			{object}	FeedPostViewModelPaginationResponse
//	@Failure		400			{object}	JSONError	"page or page_size is not a positive integer, or page_size is over 100"
//	@Failure		500			{object}	JSONError
//	@Router			/users/feed [get]
func (app *application) getFeedHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	feed, total, err := app.store.Post.GetUserFeed(r.Context(), authUser(r).ID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(feed, page, total), "feed retrieved successfully")
}
