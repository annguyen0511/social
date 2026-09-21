package main

import (
	"net/http"
)

func (app *application) getFeedHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	feed, total, err := app.store.Post.GetUserFeed(r.Context(), currentUserID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(feed, page, total), "feed retrieved successfully")
}
