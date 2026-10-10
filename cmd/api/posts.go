package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
	"github.com/annguyen0511/social/internal/upload"
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

		if app.hideWhenNotVisible(w, r, post) {
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

// parseTags turns the comma-separated line a form sends into the array the
// database stores. Empty pieces are dropped, which is what makes a trailing
// comma harmless.
//
// parseTags biến dòng ngăn bằng dấu phẩy mà form gửi lên thành mảng mà
// database lưu. Các phần rỗng bị loại, và đó là thứ khiến một dấu phẩy thừa ở
// cuối không gây lỗi.
func parseTags(value string) []string {
	tags := []string{}
	for _, tag := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

type createPostRequest struct {
	Content string   `json:"content" validate:"required,max=1000" example:"Optimistic locking is one extra predicate in the WHERE clause."`
	Title   string   `json:"title" validate:"required,max=100" example:"Optimistic locking in practice"`
	Tags    []string `json:"tags" example:"go,postgres"`
	// Visibility defaults to public when left out, so an older client that
	// does not know about this field keeps behaving the way it used to.
	//
	// Visibility mặc định là public khi không gửi, nên một client cũ chưa
	// biết tới field này vẫn hoạt động y như trước.
	Visibility string `json:"visibility" validate:"omitempty,oneof=public private" enums:"public,private" example:"public"`
} //@name PostCreateModel

// createPostHandler godoc
//
//	@Summary		Create a post
//	@Description	Creates a post owned by the signed-in user. Sent as a multipart form so an optional picture can travel with it; the picture is re-encoded, which drops any metadata it carried, and it is served back through this API rather than as a public file so that a private post keeps its picture private.
//	@Tags			Post
//	@Accept			mpfd
//	@Produce		json
//	@Param			title		formData	string	true	"Title, at most 100 characters"
//	@Param			content		formData	string	true	"Body, at most 1000 characters"
//	@Param			tags		formData	string	false	"Tags, separated by commas"
//	@Param			visibility	formData	string	false	"public or private; public when left out"	Enums(public, private)
//	@Param			image		formData	file	false	"Optional picture, at most 5 MB"
//	@Success		201			{object}	PostViewModelResponse
//	@Failure		400			{object}	JSONError
//	@Failure		500			{object}	JSONError
//	@Router			/posts [post]
func (app *application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	// The whole request is capped before anything is parsed: the image limit
	// plus room for the text fields and the multipart framing around them.
	//
	// Cả request bị chặn giới hạn trước khi phân tích bất cứ thứ gì: giới hạn
	// của ảnh cộng thêm chỗ cho các field chữ và phần khung multipart bọc
	// quanh chúng.
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxBytes+1<<20)

	// A form rather than JSON, because the picture travels with the post. Two
	// requests — create, then attach — would leave a post stranded without
	// its image whenever the second one failed, and this version has no way
	// to edit one in afterwards.
	//
	// Dùng form chứ không phải JSON, vì tấm ảnh đi cùng bài viết. Hai request
	// — tạo rồi gắn — sẽ để lại một bài trơ trọi không có ảnh mỗi khi bước
	// thứ hai hỏng, mà phiên bản này thì chưa sửa bài để thêm ảnh vào được.
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		app.badRequestResponse(w, r, errors.New("send the post as a multipart form"))
		return
	}

	req := createPostRequest{
		Title:      strings.TrimSpace(r.FormValue("title")),
		Content:    strings.TrimSpace(r.FormValue("content")),
		Tags:       parseTags(r.FormValue("tags")),
		Visibility: r.FormValue("visibility"),
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if req.Visibility == "" {
		req.Visibility = store.VisibilityPublic
	}

	// The picture is processed before the post exists, so a bad file costs
	// nothing but a rejected request.
	//
	// Ảnh được xử lý trước khi bài tồn tại, nên một file hỏng chỉ khiến
	// request bị từ chối chứ không để lại gì.
	image, err := app.readPostImage(r)
	if err != nil {
		switch {
		case errors.Is(err, upload.ErrTooLarge), errors.Is(err, upload.ErrNotAnImage):
			app.badRequestResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	post := model.Post{
		Content:    req.Content,
		Title:      req.Title,
		Tags:       req.Tags,
		Visibility: req.Visibility,
		UserID:     authUser(r).ID,
	}

	if err := app.store.Post.Create(r.Context(), &post); err != nil {
		app.discardPostImage(image)
		app.internalServerError(w, r, err)
		return
	}

	if image != nil {
		if err := app.store.Post.AttachImage(r.Context(), post.ID, image.name, image.width, image.height); err != nil {
			// The post is already in the database, so the only thing left to
			// undo is the file nothing now points at.
			//
			// Bài đã nằm trong database rồi, nên thứ duy nhất còn phải hoàn
			// tác là file mà giờ không có gì trỏ tới nữa.
			app.discardPostImage(image)
			app.internalServerError(w, r, err)
			return
		}
		post.Image = &model.PostImage{
			URL:    fmt.Sprintf("/v1/posts/%d/image", post.ID),
			Width:  image.width,
			Height: image.height,
		}
	}

	if err := app.jsonResponse(w, r, http.StatusCreated, post, "Post created successfully"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// storedImage is a picture already written to disk but not yet linked to
// anything, which is the only window where it can be orphaned.
//
// storedImage là một tấm ảnh đã ghi xuống đĩa nhưng chưa nối với thứ gì, và
// đó là khoảng thời gian duy nhất nó có thể thành file mồ côi.
type storedImage struct {
	name   string
	width  int
	height int
}

// readPostImage processes the optional "image" field and writes it to disk.
// Returns nil when the form carried no picture.
//
// readPostImage xử lý field "image" tuỳ chọn rồi ghi xuống đĩa. Trả về nil
// khi form không kèm ảnh nào.
func (app *application) readPostImage(r *http.Request) (*storedImage, error) {
	file, _, err := r.FormFile("image")
	if err != nil {
		// No file in the form is the ordinary case, not a failure.
		// Không có file trong form là trường hợp bình thường, không phải lỗi.
		return nil, nil
	}
	defer file.Close()

	data, width, height, err := upload.PostImage(file)
	if err != nil {
		return nil, err
	}

	name, err := app.postImages.Save(data)
	if err != nil {
		return nil, err
	}
	return &storedImage{name: name, width: width, height: height}, nil
}

// discardPostImage removes a file that never became part of a post, logging
// rather than failing: the caller is already on its way to reporting an
// error, and a leftover file is wasted disk, not a second thing to tell the
// user about.
//
// discardPostImage xoá một file chưa kịp thuộc về bài nào, gặp lỗi thì ghi
// log chứ không báo hỏng: phía gọi vốn đã đang trên đường báo một lỗi khác,
// mà một file sót lại chỉ là dung lượng lãng phí chứ không phải chuyện thứ
// hai cần nói với người dùng.
func (app *application) discardPostImage(image *storedImage) {
	if image == nil {
		return
	}
	if err := app.postImages.Remove(image.name); err != nil {
		app.logger.Errorw("could not remove orphaned post image", "file", image.name, "error", err)
	}
}

type updatePostRequest struct {
	Title      *string   `json:"title" validate:"omitempty,max=100" example:"An updated title"`
	Content    *string   `json:"content" validate:"omitempty,max=1000" example:"Updated content."`
	Tags       []*string `json:"tags" validate:"omitempty" example:"go,api"`
	Visibility *string   `json:"visibility" validate:"omitempty,oneof=public private" enums:"public,private" example:"private"`
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

	if req.Visibility != nil {
		post.Visibility = *req.Visibility
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

	// Read the file name before the row goes: ON DELETE CASCADE removes the
	// post_images row but never the file, and once the row is gone there is
	// nothing left to say which file it was.
	//
	// Đọc tên file trước khi dòng dữ liệu biến mất: ON DELETE CASCADE xoá
	// dòng trong post_images nhưng không bao giờ xoá file, mà dòng đã mất
	// rồi thì không còn gì cho biết đó là file nào.
	imageName, err := app.store.Post.ImageName(r.Context(), post.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.store.Post.Delete(r.Context(), post.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if imageName != "" {
		app.discardPostImage(&storedImage{name: imageName})
	}
}

// servePostImageHandler serves a post's picture.
//
// It sits under /posts/{postID}, which is the whole point: postContextMiddileware
// has already turned a blocked author or a private post into a 404 before
// this runs, so the picture is exactly as reachable as the post it belongs
// to, with no second copy of that logic to keep in step.
//
// That is also why it cannot be a plain static file the way an avatar is. An
// avatar is public; a picture on a close-friends post is not, and a file
// served straight off disk has no idea who is asking.
//
// servePostImageHandler phục vụ ảnh của một bài viết.
//
// Nó nằm dưới /posts/{postID}, và đó chính là toàn bộ ý đồ:
// postContextMiddileware đã biến tác giả bị chặn hay bài riêng tư thành 404
// từ trước khi hàm này chạy, nên tấm ảnh dễ tiếp cận đúng bằng bài viết chứa
// nó, mà không có bản sao thứ hai nào của logic đó phải giữ cho khớp.
//
// Đó cũng là lý do nó không thể là file tĩnh như avatar. Avatar là công
// khai; ảnh của một bài chỉ dành cho bạn thân thì không, mà file phục vụ
// thẳng từ đĩa thì chẳng biết ai đang hỏi.
func (app *application) servePostImageHandler(w http.ResponseWriter, r *http.Request) {
	post, ok := getPostFromContext(r)
	if !ok {
		app.internalServerError(w, r, errPostMissing)
		return
	}

	name, err := app.store.Post.ImageName(r.Context(), post.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if name == "" {
		app.notFoundResponse(w, r, errors.New("the post has no image"))
		return
	}

	file, ok := app.postImages.Path(name)
	if !ok {
		app.notFoundResponse(w, r, errors.New("no such image"))
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// private, not public: a shared cache must not keep a copy that it would
	// hand to someone the post was never meant for. immutable holds because
	// this version does not let a post's picture be replaced.
	//
	// private chứ không phải public: một cache dùng chung không được giữ bản
	// sao rồi đưa cho người mà bài viết vốn không dành cho. immutable đúng vì
	// phiên bản này chưa cho thay ảnh của một bài đã đăng.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")

	http.ServeFile(w, r, file)
}
