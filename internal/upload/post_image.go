package upload

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"

	"golang.org/x/image/draw"
)

// PostSize is the longest side a stored post image is allowed to have.
//
// It is a bound, not a target: a picture smaller than this is left at its own
// size rather than blown up, because enlarging only invents pixels and makes
// the file bigger.
//
// PostSize là cạnh dài nhất mà một ảnh bài viết được phép có khi lưu.
//
// Đây là giới hạn trên chứ không phải kích thước đích: ảnh nhỏ hơn mức này
// được giữ nguyên chứ không phóng to, vì phóng to chỉ bịa thêm điểm ảnh và
// làm file nặng lên.
const PostSize = 1080

// PostImage reads an uploaded file and returns a JPEG together with the size
// it ended up.
//
// Unlike Avatar it does not crop. An avatar is always square so forcing the
// shape is right; a photo is whatever shape it was taken in, and cropping it
// to fit would cut people out of their own pictures.
//
// It decodes and re-encodes for the same three reasons Avatar does, and the
// first matters more here: a photo straight off a phone carries the GPS
// coordinates of where it was taken, and a post is seen by more people than
// an avatar ever is.
//
// PostImage đọc file được tải lên và trả về ảnh JPEG cùng kích thước cuối.
//
// Khác với Avatar, nó không cắt. Avatar luôn vuông nên ép khuôn là đúng; còn
// một tấm ảnh thì có hình dạng vốn có của nó, cắt cho vừa khung là cắt mất
// người ra khỏi chính bức ảnh của họ.
//
// Nó giải mã rồi mã hoá lại vì đúng ba lý do như Avatar, và lý do đầu ở đây
// còn quan trọng hơn: ảnh vừa chụp từ điện thoại mang theo toạ độ GPS nơi
// chụp, mà một bài viết thì nhiều người xem hơn avatar nhiều.
func PostImage(r io.Reader) (data []byte, width, height int, err error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return nil, 0, 0, err
	}
	if len(raw) > MaxBytes {
		return nil, 0, 0, ErrTooLarge
	}
	if len(raw) == 0 {
		return nil, 0, 0, ErrNotAnImage
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, ErrNotAnImage
	}
	if config.Width > maxDimension || config.Height > maxDimension {
		return nil, 0, 0, ErrTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, ErrNotAnImage
	}

	bounds := src.Bounds()
	width, height = fitWithin(bounds.Dx(), bounds.Dy(), PostSize)

	out := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(out, out.Bounds(), src, bounds, draw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: 85}); err != nil {
		return nil, 0, 0, fmt.Errorf("encode post image: %w", err)
	}
	return buf.Bytes(), width, height, nil
}

// fitWithin scales w by h down so neither side passes limit, keeping the
// ratio. Anything already inside the limit comes back unchanged.
//
// The rounding is deliberate: scaling both sides by the same float and
// truncating can land on 0 for a very thin image, and an image of zero width
// cannot be encoded.
//
// fitWithin thu nhỏ w nhân h sao cho không cạnh nào vượt limit, giữ nguyên tỉ
// lệ. Ảnh vốn đã nằm trong giới hạn thì trả về y nguyên.
//
// Việc làm tròn là có chủ đích: nhân cả hai cạnh với cùng một số thực rồi cắt
// phần thập phân có thể ra 0 với một tấm ảnh rất mảnh, mà ảnh rộng 0 điểm thì
// không mã hoá được.
func fitWithin(w, h, limit int) (int, int) {
	if w <= limit && h <= limit {
		return w, h
	}

	if w >= h {
		return limit, max(1, h*limit/w)
	}
	return max(1, w*limit/h), limit
}
