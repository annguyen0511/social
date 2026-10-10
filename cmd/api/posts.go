package main

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
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
//	@Description	Creates a post owned by the signed-in user. Sent as a multipart form so up to 10 pictures can travel with it; each picture is re-encoded, which drops any metadata it carried, and they are served back through this API rather than as public files so that a private post keeps its pictures private. The pictures keep the order the form lists them in.
//	@Tags			Post
//	@Accept			mpfd
//	@Produce		json
//	@Param			title		formData	string	true	"Title, at most 100 characters"
//	@Param			content		formData	string	true	"Body, at most 1000 characters"
//	@Param			tags		formData	string	false	"Tags, separated by commas"
//	@Param			visibility	formData	string	false	"public or private; public when left out"	Enums(public, private)
//	@Param			images		formData	file	false	"Optional pictures, at most 10 of them and 5 MB each. Repeat the field once per picture."
//	@Param			crop_x		formData	int		false	"Crop origin X, in the uploaded image's own pixels. Send all four crop fields once per picture, in the same order as the files, or none at all."
//	@Param			crop_y		formData	int		false	"Crop origin Y"
//	@Param			crop_width	formData	int		false	"Crop width"
//	@Param			crop_height	formData	int		false	"Crop height"
//	@Success		201			{object}	PostViewModelResponse
//	@Failure		400			{object}	JSONError
//	@Failure		500			{object}	JSONError
//	@Router			/posts [post]
func (app *application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	// The whole request is capped before anything is parsed: every picture a
	// post may carry, plus room for the text fields and the multipart
	// framing around them.
	//
	// Cả request bị chặn giới hạn trước khi phân tích bất cứ thứ gì: toàn bộ
	// số ảnh một bài được mang, cộng thêm chỗ cho các field chữ và phần khung
	// multipart bọc quanh chúng.
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxBytes*maxImages+1<<20)

	// A form rather than JSON, because the pictures travel with the post. Two
	// requests — create, then attach — would leave a post stranded without
	// its images whenever the second one failed, and this version has no way
	// to edit them in afterwards.
	//
	// Dùng form chứ không phải JSON, vì những tấm ảnh đi cùng bài viết. Hai
	// request — tạo rồi gắn — sẽ để lại một bài trơ trọi không có ảnh mỗi khi
	// bước thứ hai hỏng, mà phiên bản này thì chưa sửa bài để thêm ảnh vào
	// được.
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

	// The pictures are processed before the post exists, so a bad file costs
	// nothing but a rejected request.
	//
	// Ảnh được xử lý trước khi bài tồn tại, nên một file hỏng chỉ khiến
	// request bị từ chối chứ không để lại gì.
	images, err := app.readPostImages(r)
	if err != nil {
		switch {
		case errors.Is(err, upload.ErrTooLarge), errors.Is(err, upload.ErrNotAnImage),
			errors.Is(err, upload.ErrBadCrop), errors.Is(err, errTooManyImages):
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
		app.discardPostImages(images)
		app.internalServerError(w, r, err)
		return
	}

	// Empty rather than nil, to match what every read path returns: the
	// client maps over this field without checking it first.
	//
	// Rỗng chứ không phải nil, để khớp với thứ mọi đường đọc trả về: client
	// duyệt qua field này mà không kiểm trước.
	post.Images = make([]model.PostImage, 0, len(images))

	// The position is the index in the form, which is the order the user
	// arranged them in; it is what every read path sorts by.
	//
	// position là chỉ số trong form, cũng là thứ tự người dùng đã sắp; đó là
	// thứ mà mọi đường đọc đều sắp xếp theo.
	for i, image := range images {
		if err := app.store.Post.AttachImage(r.Context(), post.ID, image.name, image.width, image.height, i); err != nil {
			// The post is already in the database, so what is left to undo is
			// the files nothing points at — the ones not yet attached. Those
			// already linked now belong to the post and go when it does.
			//
			// Bài đã nằm trong database rồi, nên thứ còn phải hoàn tác là
			// những file không có gì trỏ tới — tức là các file chưa gắn được.
			// Những file đã nối xong giờ thuộc về bài và sẽ đi cùng nó.
			app.discardPostImages(images[i:])
			app.internalServerError(w, r, err)
			return
		}
		post.Images = append(post.Images, model.PostImage{
			URL:    fmt.Sprintf("/v1/posts/%d/image/%s", post.ID, image.name),
			Width:  image.width,
			Height: image.height,
		})
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

// maxImages caps one post. Ten is where Instagram stops and it is a sensible
// place to stop too: each file may be upload.MaxBytes, so this number is also
// what bounds the size of a single request.
//
// maxImages giới hạn một bài. Mười là mức Instagram dừng lại và cũng là chỗ
// dừng hợp lý: mỗi file có thể tới upload.MaxBytes, nên con số này cũng chính
// là thứ chặn kích thước của một request.
const maxImages = 10

var errTooManyImages = fmt.Errorf("a post can carry at most %d pictures", maxImages)

// readPostImages processes the optional "images" fields and writes them to
// disk in the order the form listed them. Returns nil when the form carried
// no pictures.
//
// readPostImages xử lý các field "images" tuỳ chọn rồi ghi xuống đĩa theo
// đúng thứ tự form liệt kê. Trả về nil khi form không kèm ảnh nào.
func (app *application) readPostImages(r *http.Request) ([]storedImage, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		return nil, nil
	}
	if len(files) > maxImages {
		return nil, errTooManyImages
	}

	crops, err := readCropRects(r, len(files))
	if err != nil {
		return nil, err
	}

	stored := make([]storedImage, 0, len(files))
	for i, header := range files {
		image, err := app.readOneImage(header, crops[i])
		if err != nil {
			// Whatever reached the disk before the failure belongs to nobody,
			// so none of it is kept: the request is about to be rejected
			// whole.
			//
			// Thứ đã kịp xuống đĩa trước khi hỏng thì không thuộc về ai, nên
			// không giữ lại gì cả: cả request sắp bị từ chối.
			app.discardPostImages(stored)
			return nil, err
		}
		stored = append(stored, image)
	}
	return stored, nil
}

func (app *application) readOneImage(header *multipart.FileHeader, crop *upload.CropRect) (storedImage, error) {
	file, err := header.Open()
	if err != nil {
		return storedImage{}, err
	}
	defer file.Close()

	data, width, height, err := upload.PostImage(file, crop)
	if err != nil {
		return storedImage{}, err
	}

	name, err := app.postImages.Save(data)
	if err != nil {
		return storedImage{}, err
	}
	return storedImage{name: name, width: width, height: height}, nil
}

// readCropRects reads the crop the browser measured for each picture, or
// nothing when the form carried none.
//
// All four fields or none, and each one repeated once per picture: three
// values describe no rectangle at all, and a list shorter than the pictures
// could only be lined up by guessing which photo each crop belongs to —
// guess wrong and the wrong photo gets cut. Silently treating either as "no
// crop" would quietly post whole pictures when the client meant to send
// parts of them. Better to say the form is wrong.
//
// readCropRects đọc vùng cắt mà trình duyệt đã đo cho từng ảnh, hoặc không gì
// cả khi form không kèm theo.
//
// Hoặc đủ bốn field hoặc không field nào, và mỗi field lặp lại đúng một lần
// cho mỗi ảnh: ba giá trị thì không mô tả nổi một hình chữ nhật, còn một danh
// sách ngắn hơn số ảnh thì chỉ ghép lại được bằng cách đoán vùng cắt nào của
// tấm nào — đoán sai là cắt nhầm ảnh. Âm thầm coi cả hai là "không cắt" sẽ
// lặng lẽ đăng nguyên những tấm ảnh trong khi client có ý gửi một phần của
// chúng. Thà báo là form sai.
func readCropRects(r *http.Request, count int) ([]*upload.CropRect, error) {
	keys := []string{"crop_x", "crop_y", "crop_width", "crop_height"}
	columns := make([][]string, len(keys))

	present := 0
	for i, key := range keys {
		columns[i] = r.MultipartForm.Value[key]
		if len(columns[i]) > 0 {
			present++
		}
	}
	if present == 0 {
		// A nil crop per picture means "use the whole thing".
		// Mỗi ảnh một vùng cắt nil nghĩa là "dùng nguyên tấm".
		return make([]*upload.CropRect, count), nil
	}
	if present != len(keys) {
		return nil, fmt.Errorf("%w: send all of crop_x, crop_y, crop_width and crop_height, or none", upload.ErrBadCrop)
	}
	for i, key := range keys {
		if len(columns[i]) != count {
			return nil, fmt.Errorf("%w: send %s once per picture, in the same order", upload.ErrBadCrop, key)
		}
	}

	crops := make([]*upload.CropRect, count)
	for row := range count {
		values := make([]int, len(keys))
		for i, key := range keys {
			n, err := strconv.Atoi(columns[i][row])
			if err != nil {
				return nil, fmt.Errorf("%w: %s must be a whole number", upload.ErrBadCrop, key)
			}
			values[i] = n
		}

		if values[2] <= 0 || values[3] <= 0 {
			return nil, fmt.Errorf("%w: crop_width and crop_height must be positive", upload.ErrBadCrop)
		}

		crops[row] = &upload.CropRect{X: values[0], Y: values[1], Width: values[2], Height: values[3]}
	}
	return crops, nil
}

// discardPostImages removes files that never became part of a post, logging
// rather than failing: the caller is already on its way to reporting an
// error, and a leftover file is wasted disk, not a second thing to tell the
// user about.
//
// discardPostImages xoá những file chưa kịp thuộc về bài nào, gặp lỗi thì ghi
// log chứ không báo hỏng: phía gọi vốn đã đang trên đường báo một lỗi khác,
// mà một file sót lại chỉ là dung lượng lãng phí chứ không phải chuyện thứ
// hai cần nói với người dùng.
func (app *application) discardPostImages(images []storedImage) {
	for _, image := range images {
		if err := app.postImages.Remove(image.name); err != nil {
			app.logger.Errorw("could not remove orphaned post image", "file", image.name, "error", err)
		}
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

	// Read the file names before the rows go: ON DELETE CASCADE removes the
	// post_images rows but never the files, and once the rows are gone there
	// is nothing left to say which files they were.
	//
	// Đọc tên file trước khi các dòng dữ liệu biến mất: ON DELETE CASCADE xoá
	// các dòng trong post_images nhưng không bao giờ xoá file, mà dòng đã mất
	// rồi thì không còn gì cho biết đó là những file nào.
	names, err := app.store.Post.ImageNames(r.Context(), post.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.store.Post.Delete(r.Context(), post.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	images := make([]storedImage, 0, len(names))
	for _, name := range names {
		images = append(images, storedImage{name: name})
	}
	app.discardPostImages(images)
}

// servePostImageHandler godoc
//
//	@Summary		Serve one of a post's pictures
//	@Description	Serves a single picture by the file name the post's images list gives. The picture is exactly as reachable as the post itself.
//	@Tags			Post
//	@Produce		jpeg
//	@Param			postID		path	int		true	"Post ID"
//	@Param			imageName	path	string	true	"File name, from the post's images list"
//	@Success		200
//	@Failure		404	{object}	JSONError
//	@Failure		500	{object}	JSONError
//	@Router			/posts/{postID}/image/{imageName} [get]
//
// servePostImageHandler serves one of a post's pictures.
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
// servePostImageHandler phục vụ một trong các ảnh của bài viết.
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

	// The name comes from the URL, so it is checked against this post rather
	// than trusted: Path rejects anything that is not one of our generated
	// names, and HasImage makes sure it is a picture of *this* post and not
	// of another one the viewer may not be allowed to see.
	//
	// Tên lấy từ URL nên phải đối chiếu với bài này chứ không tin ngay: Path
	// loại mọi thứ không phải tên do ta sinh ra, còn HasImage bảo đảm đó là
	// ảnh của *bài này*, không phải của một bài khác mà người xem có thể
	// không được phép xem.
	name := chi.URLParam(r, "imageName")

	ok, err := app.store.Post.HasImage(r.Context(), post.ID, name)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if !ok {
		app.notFoundResponse(w, r, errors.New("no such image on this post"))
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
