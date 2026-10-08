package main

import (
	"errors"
	"net/http"

	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
)

// friendshipTarget returns the user the action is aimed at, already loaded by
// userContextMiddleware, and rejects actions a user aims at themselves.
func (app *application) friendshipTarget(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	target, ok := getUserFromContext(r)
	if !ok {
		app.internalServerError(w, r, errors.New("user missing from request context"))
		return nil, false
	}

	if target.ID == authUser(r).ID {
		app.badRequestResponse(w, r, errors.New("a user cannot perform this action on themselves"))
		return nil, false
	}

	return target, true
}

// followUserHandler godoc
//
//	@Summary		Follow a user
//	@Description	Idempotent: following someone already followed still returns 200. Rejected with 403 while either user blocks the other.
//	@Tags			Friendship
//	@Produce		json
//	@Param			userID	path		int	true	"ID of the user the action is aimed at, never the actor"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	JSONError	"userID is not a number, or is the current user"
//	@Failure		403		{object}	JSONError	"A block exists between the two users"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/friend-ship/{userID}/follow [put]
func (app *application) followUserHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.Follow.Follow(r.Context(), authUser(r).ID, target.ID); err != nil {
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

// unfollowUserHandler godoc
//
//	@Summary		Unfollow a user
//	@Description	Idempotent: unfollowing someone not followed still returns 200.
//	@Tags			Friendship
//	@Produce		json
//	@Param			userID	path		int	true	"ID of the user the action is aimed at, never the actor"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	JSONError	"userID is not a number, or is the current user"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/friend-ship/{userID}/follow [delete]
func (app *application) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.Follow.Unfollow(r.Context(), authUser(r).ID, target.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "user unfollowed successfully")
}

// blockUserHandler godoc
//
//	@Summary		Block a user
//	@Description	Severs the relationship in both directions in one transaction: removes follows and close friend entries either way, then records the block. Unblocking later does not restore them.
//	@Tags			Friendship
//	@Produce		json
//	@Param			userID	path		int	true	"ID of the user the action is aimed at, never the actor"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	JSONError	"userID is not a number, or is the current user"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/friend-ship/{userID}/block [put]
func (app *application) blockUserHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.Block.Block(r.Context(), authUser(r).ID, target.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "user blocked successfully")
}

// unblockUserHandler godoc
//
//	@Summary		Unblock a user
//	@Description	Lifts the block only. Follows removed by the block are not restored.
//	@Tags			Friendship
//	@Produce		json
//	@Param			userID	path		int	true	"ID of the user the action is aimed at, never the actor"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	JSONError	"userID is not a number, or is the current user"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/friend-ship/{userID}/block [delete]
func (app *application) unblockUserHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.Block.Unblock(r.Context(), authUser(r).ID, target.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, nil, "user unblocked successfully")
}

// addCloseFriendHandler godoc
//
//	@Summary		Add a close friend
//	@Description	Idempotent. Rejected with 403 while either user blocks the other.
//	@Tags			Friendship
//	@Produce		json
//	@Param			userID	path		int	true	"ID of the user the action is aimed at, never the actor"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	JSONError	"userID is not a number, or is the current user"
//	@Failure		403		{object}	JSONError	"A block exists between the two users"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/friend-ship/{userID}/close-friend [put]
func (app *application) addCloseFriendHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.CloseFriend.Add(r.Context(), authUser(r).ID, target.ID); err != nil {
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

// removeCloseFriendHandler godoc
//
//	@Summary		Remove a close friend
//	@Description	Idempotent.
//	@Tags			Friendship
//	@Produce		json
//	@Param			userID	path		int	true	"ID of the user the action is aimed at, never the actor"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	JSONError	"userID is not a number, or is the current user"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/friend-ship/{userID}/close-friend [delete]
func (app *application) removeCloseFriendHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	if err := app.store.CloseFriend.Remove(r.Context(), authUser(r).ID, target.ID); err != nil {
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
} //@name FriendshipStatusViewModel

// friendshipStatusHandler reports how the current user relates to the user in
// the route.
// friendshipStatusHandler godoc
//
//	@Summary		Get friendship status
//	@Description	How the current user relates to the user in the route.
//	@Tags			Friendship
//	@Produce		json
//	@Param			userID	path		int	true	"ID of the user the action is aimed at, never the actor"
//	@Success		200		{object}	FriendshipStatusViewModelResponse
//	@Failure		400		{object}	JSONError	"userID is not a number, or is the current user"
//	@Failure		404		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/friend-ship/{userID} [get]
func (app *application) friendshipStatusHandler(w http.ResponseWriter, r *http.Request) {
	target, ok := app.friendshipTarget(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	isFollowing, err := app.store.Follow.IsFollowing(ctx, authUser(r).ID, target.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	isBlocking, err := app.store.Block.IsBlocking(ctx, authUser(r).ID, target.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	isCloseFriend, err := app.store.CloseFriend.IsCloseFriend(ctx, authUser(r).ID, target.ID)
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

// listFollowersHandler godoc
//
//	@Summary	List my followers
//	@Tags		Friendship
//	@Produce	json
//	@Param		page		query		int	false	"Page number, starting at 1"	default(1)	minimum(1)
//	@Param		page_size	query		int	false	"Items per page"				default(20)	minimum(1)	maximum(100)
//	@Success	200			{object}	FollowViewModelPaginationResponse
//	@Failure	400			{object}	JSONError	"page or page_size is not a positive integer, or page_size is over 100"
//	@Failure	500			{object}	JSONError
//	@Router		/friend-ship/followers [get]
func (app *application) listFollowersHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	followers, total, err := app.store.Follow.GetFollowers(r.Context(), authUser(r).ID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(followers, page, total), "followers retrieved successfully")
}

// listFollowingHandler godoc
//
//	@Summary	List who I follow
//	@Tags		Friendship
//	@Produce	json
//	@Param		page		query		int	false	"Page number, starting at 1"	default(1)	minimum(1)
//	@Param		page_size	query		int	false	"Items per page"				default(20)	minimum(1)	maximum(100)
//	@Success	200			{object}	FollowViewModelPaginationResponse
//	@Failure	400			{object}	JSONError	"page or page_size is not a positive integer, or page_size is over 100"
//	@Failure	500			{object}	JSONError
//	@Router		/friend-ship/following [get]
func (app *application) listFollowingHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	following, total, err := app.store.Follow.GetFollowing(r.Context(), authUser(r).ID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(following, page, total), "following retrieved successfully")
}

// listBlockingHandler godoc
//
//	@Summary	List users I block
//	@Tags		Friendship
//	@Produce	json
//	@Param		page		query		int	false	"Page number, starting at 1"	default(1)	minimum(1)
//	@Param		page_size	query		int	false	"Items per page"				default(20)	minimum(1)	maximum(100)
//	@Success	200			{object}	BlockViewModelPaginationResponse
//	@Failure	400			{object}	JSONError	"page or page_size is not a positive integer, or page_size is over 100"
//	@Failure	500			{object}	JSONError
//	@Router		/friend-ship/blocking [get]
func (app *application) listBlockingHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	blocking, total, err := app.store.Block.ListBlocking(r.Context(), authUser(r).ID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(blocking, page, total), "blocked users retrieved successfully")
}

// listCloseFriendsHandler godoc
//
//	@Summary	List my close friends (user IDs)
//	@Tags		Friendship
//	@Produce	json
//	@Param		page		query		int	false	"Page number, starting at 1"	default(1)	minimum(1)
//	@Param		page_size	query		int	false	"Items per page"				default(20)	minimum(1)	maximum(100)
//	@Success	200			{object}	UserIDPaginationResponse
//	@Failure	400			{object}	JSONError	"page or page_size is not a positive integer, or page_size is over 100"
//	@Failure	500			{object}	JSONError
//	@Router		/friend-ship/close-friends [get]
func (app *application) listCloseFriendsHandler(w http.ResponseWriter, r *http.Request) {
	page, err := readPagination(r)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	friendIDs, total, err := app.store.CloseFriend.List(r.Context(), authUser(r).ID, page)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, r, http.StatusOK, newPagination(friendIDs, page, total), "close friends retrieved successfully")
}
