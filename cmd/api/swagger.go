package main

import "github.com/annguyen0511/social/internal/model"

// Named instantiations of Response and Pagination, used only by the Swagger
// annotations. Each gives the spec a named schema such as PostViewModelResponse,
// where writing Response[model.Post] directly would produce
// "main.Response-model_Post". They share the generic types' fields, so they
// cannot drift from what the server sends.

type MessageResponse Response[any] //@name MessageResponse

type UserRegisteredViewModelResponse Response[registeredUser] //@name UserRegisteredViewModelResponse

type PostViewModelResponse Response[model.Post]                   //@name PostViewModelResponse
type UserViewModelResponse Response[model.User]                   //@name UserViewModelResponse
type UserProfileViewModelResponse Response[userProfile]           //@name UserProfileViewModelResponse
type FriendshipStatusViewModelResponse Response[friendshipStatus] //@name FriendshipStatusViewModelResponse
type LikeViewModelResponse Response[likeState]                    //@name LikeViewModelResponse

type FeedPostViewModelPagination Pagination[model.FeedPost]       //@name FeedPostViewModelPagination
type FollowViewModelPagination Pagination[model.Follow]           //@name FollowViewModelPagination
type UserViewModelPagination Pagination[model.User]               //@name UserViewModelPagination
type UserSummaryViewModelPagination Pagination[model.UserSummary] //@name UserSummaryViewModelPagination
type UserIDPagination Pagination[int64]                           //@name UserIDPagination

type FeedPostViewModelPaginationResponse Response[FeedPostViewModelPagination]       //@name FeedPostViewModelPaginationResponse
type FollowViewModelPaginationResponse Response[FollowViewModelPagination]           //@name FollowViewModelPaginationResponse
type UserViewModelPaginationResponse Response[UserViewModelPagination]               //@name UserViewModelPaginationResponse
type UserSummaryViewModelPaginationResponse Response[UserSummaryViewModelPagination] //@name UserSummaryViewModelPaginationResponse
type UserIDPaginationResponse Response[UserIDPagination]                             //@name UserIDPaginationResponse
