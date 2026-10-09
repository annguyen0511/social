package main

import (
	"errors"
	"net/http"
	"path"

	"github.com/annguyen0511/social/internal/store"
	"github.com/annguyen0511/social/internal/upload"
	"github.com/go-chi/chi/v5"
)

// avatarURLPrefix is what gets stored in users.avatar_url, so the column holds
// a path the browser can request directly. It starts with /v1 so the one proxy
// rule the frontend already has covers it in development.
//
// avatarURLPrefix là thứ được lưu vào users.avatar_url, nên cột đó chứa một
// đường dẫn mà trình duyệt yêu cầu được ngay. Nó bắt đầu bằng /v1 để một quy
// tắc proxy duy nhất mà frontend vốn đã có là đủ bao lúc dev.
const avatarURLPrefix = "/v1/uploads/avatars/"

// uploadAvatarHandler godoc
//
//	@Summary		Upload my avatar
//	@Description	Accepts a multipart form with one file field named "avatar". The image is re-encoded to a 256x256 JPEG, which drops any metadata it carried, and the previous picture is deleted. Maximum 5 MB; JPEG, PNG and GIF are accepted.
//	@Tags			User
//	@Accept			mpfd
//	@Produce		json
//	@Param			avatar	formData	file	true	"Image file"
//	@Success		200		{object}	UserViewModelResponse
//	@Failure		400		{object}	JSONError	"No file sent, the file is not an image, or it is too large"
//	@Failure		401		{object}	JSONError
//	@Failure		500		{object}	JSONError
//	@Router			/users/me/avatar [post]
func (app *application) uploadAvatarHandler(w http.ResponseWriter, r *http.Request) {
	// The reader is capped before anything is parsed. upload.Avatar enforces
	// the same limit on the image itself; this covers the multipart envelope
	// around it, so a request cannot be made enormous by its own framing.
	//
	// Luồng đọc bị chặn giới hạn trước khi phân tích bất cứ thứ gì.
	// upload.Avatar áp đúng giới hạn đó lên bản thân tấm ảnh; chỗ này lo phần
	// vỏ multipart bọc quanh, để một request không thể phình to bằng chính
	// phần khung của nó.
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxBytes+1<<20)

	file, _, err := r.FormFile("avatar")
	if err != nil {
		app.badRequestResponse(w, r, errors.New("send one image in a form field named avatar"))
		return
	}
	defer file.Close()

	data, err := upload.Avatar(file)
	if err != nil {
		switch {
		case errors.Is(err, upload.ErrTooLarge), errors.Is(err, upload.ErrNotAnImage):
			app.badRequestResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	name, err := app.avatars.Save(data)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	user := *authUser(r)
	previous := user.AvatarURL
	user.AvatarURL = avatarURLPrefix + name

	if err := app.store.User.Update(r.Context(), &user); err != nil {
		// The row still points at the old picture, so the file just written
		// belongs to nobody. Clean it up rather than leave it on disk forever.
		//
		// Dòng dữ liệu vẫn đang trỏ tới ảnh cũ, nên file vừa ghi không thuộc
		// về ai cả. Dọn nó đi thay vì để nằm lại trên đĩa mãi mãi.
		if rmErr := app.avatars.Remove(user.AvatarURL); rmErr != nil {
			app.logger.Errorw("could not remove orphaned avatar", "file", name, "error", rmErr)
		}

		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	// Only once the row points at the new file is the old one safe to delete.
	// Doing it earlier would leave the avatar broken if the update failed.
	//
	// Chỉ khi dòng dữ liệu đã trỏ sang file mới thì mới an toàn để xoá file
	// cũ. Xoá sớm hơn sẽ khiến avatar hỏng nếu lệnh cập nhật thất bại.
	app.removeAvatarFile(previous)

	if err := app.jsonResponse(w, r, http.StatusOK, user, "avatar updated successfully"); err != nil {
		app.internalServerError(w, r, err)
	}
}

// deleteAvatarHandler godoc
//
//	@Summary		Remove my avatar
//	@Description	Clears the picture and deletes the stored file. The interface falls back to the user's initials. Calling it with no avatar set is not an error.
//	@Tags			User
//	@Produce		json
//	@Success		200	{object}	UserViewModelResponse
//	@Failure		401	{object}	JSONError
//	@Failure		500	{object}	JSONError
//	@Router			/users/me/avatar [delete]
func (app *application) deleteAvatarHandler(w http.ResponseWriter, r *http.Request) {
	user := *authUser(r)
	previous := user.AvatarURL
	user.AvatarURL = ""

	if err := app.store.User.Update(r.Context(), &user); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.removeAvatarFile(previous)

	if err := app.jsonResponse(w, r, http.StatusOK, user, "avatar removed successfully"); err != nil {
		app.internalServerError(w, r, err)
	}
}

// removeAvatarFile deletes a stored picture, logging rather than failing. The
// database already says the avatar is gone, so a leftover file is wasted disk,
// not a broken response for the caller.
//
// removeAvatarFile xoá một ảnh đã lưu, gặp lỗi thì ghi log chứ không báo hỏng.
// Database đã ghi nhận avatar không còn, nên một file sót lại chỉ là dung
// lượng lãng phí, không phải một response hỏng với người gọi.
func (app *application) removeAvatarFile(avatarURL string) {
	if avatarURL == "" {
		return
	}
	if err := app.avatars.Remove(avatarURL); err != nil {
		app.logger.Errorw("could not remove old avatar", "url", avatarURL, "error", err)
	}
}

// serveAvatarHandler serves a stored avatar. It is deliberately outside
// requireAuth: an <img> tag has to load without the page arranging anything,
// and the names are random, so one avatar's address reveals nothing else.
//
// serveAvatarHandler phục vụ một avatar đã lưu. Nó cố tình nằm ngoài
// requireAuth: một thẻ <img> phải tải được mà không cần trang sắp xếp gì thêm,
// và tên file là ngẫu nhiên nên địa chỉ của một avatar không tiết lộ gì khác.
func (app *application) serveAvatarHandler(w http.ResponseWriter, r *http.Request) {
	name := path.Base(chi.URLParam(r, "name"))

	file, ok := app.avatars.Path(name)
	if !ok {
		app.notFoundResponse(w, r, errors.New("no such avatar"))
		return
	}

	// nosniff keeps a browser from deciding for itself what the bytes are.
	// The content is re-encoded JPEG so this should never matter, and that is
	// exactly why it costs nothing to say so.
	//
	// nosniff ngăn trình duyệt tự suy đoán xem đống byte này là gì. Nội dung
	// đã được mã hoá lại thành JPEG nên lẽ ra chuyện đó không bao giờ xảy ra,
	// và chính vì vậy việc khai báo thêm chẳng tốn gì.
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// A name is only ever written once and never reused, so the bytes behind
	// it can never change. Changing the picture produces a new name.
	//
	// Mỗi cái tên chỉ được ghi một lần và không bao giờ dùng lại, nên nội dung
	// phía sau nó không thể đổi. Đổi ảnh sẽ sinh ra một cái tên mới.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	http.ServeFile(w, r, file)
}
