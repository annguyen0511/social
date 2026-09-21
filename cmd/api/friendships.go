package main

import (
	"errors"
	"net/http"

	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
)

// currentUserID stands in for the authenticated user until auth is implemented
const currentUserID int64 = 1

// friendshipTarget returns the user the action is aimed at, already loaded by
// userContextMiddleware, and rejects actions a user aims at themselves.
func (app *application) friendshipTarget(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	target, ok := getUserFromContext(r)
	if !ok {
		app.internalServerError(w, r, errors.New("user missing from request context"))
		return nil, false
	}

	if target.ID == currentUserID {
		app.badRequestResponse(w, r, errors.New("a user cannot perform this action on themselves"))
		return nil, false
	}

	return target, true
}

func (app *application) followUserHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.Follow.Follow(r.Context(), currentUserID, target.ID); err != nil {
		switch {
		case errors.Is(err, store.ErrBlocked):
			app.forbiddenResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "user followed successfully")
}

func (app *application) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.Follow.Unfollow(r.Context(), currentUserID, target.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "user unfollowed successfully")
}

func (app *application) blockUserHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.Block.Block(r.Context(), currentUserID, target.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "user blocked successfully")
}

func (app *application) unblockUserHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.Block.Unblock(r.Context(), currentUserID, target.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "user unblocked successfully")
}

func (app *application) addCloseFriendHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.CloseFriend.Add(r.Context(), currentUserID, target.ID); err != nil {
		switch {
		case errors.Is(err, store.ErrBlocked):
			app.forbiddenResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "close friend added successfully")
}

func (app *application) removeCloseFriendHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.CloseFriend.Remove(r.Context(), currentUserID, target.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "close friend removed successfully")
}

type friendshipStatus struct {
	UserID        int64 `json:"user_id"`
	IsFollowing   bool  `json:"is_following"`
	IsBlocking    bool  `json:"is_blocking"`
	IsCloseFriend bool  `json:"is_close_friend"`
}

// friendshipStatusHandler reports how the current user relates to the user in
// the route.
func (app *application) friendshipStatusHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	isFollowing, err := app.store.Follow.IsFollowing(ctx, currentUserID, target.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	isBlocking, err := app.store.Block.IsBlocking(ctx, currentUserID, target.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	isCloseFriend, err := app.store.CloseFriend.IsCloseFriend(ctx, currentUserID, target.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	status := friendshipStatus{
		UserID:        target.ID,
		IsFollowing:   isFollowing,
		IsBlocking:    isBlocking,
		IsCloseFriend: isCloseFriend,
	}

	app.jsonResponse(w, r, http.StatusOK, status, "friendship status retrieved successfully")
}

// The list handlers below describe the current user, so they carry no {userID}.

func (app *application) listFollowersHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	followers, total, err := app.store.Follow.GetFollowers(r.Context(), currentUserID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(followers, page, total), "followers retrieved successfully")
}

func (app *application) listFollowingHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	following, total, err := app.store.Follow.GetFollowing(r.Context(), currentUserID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(following, page, total), "following retrieved successfully")
}

func (app *application) listBlockingHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	blocking, total, err := app.store.Block.ListBlocking(r.Context(), currentUserID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(blocking, page, total), "blocked users retrieved successfully")
}

func (app *application) listCloseFriendsHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	friendIDs, total, err := app.store.CloseFriend.List(r.Context(), currentUserID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(friendIDs, page, total), "close friends retrieved successfully")
}
