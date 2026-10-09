// Package upload turns a file a user chose on their own machine into an image
// this server is willing to store and serve.
//
// Package upload biến một file mà người dùng chọn trên máy họ thành một ảnh
// mà server này chấp nhận lưu và phục vụ lại.
package upload

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"

	// Registering the decoders is what lets image.Decode recognise these
	// formats. Without the blank imports it only knows the ones already
	// linked in, and a valid PNG would be reported as "unknown format".
	//
	// Đăng ký decoder chính là thứ giúp image.Decode nhận ra các định dạng
	// này. Thiếu import trống thì nó chỉ biết những định dạng đã được liên
	// kết sẵn, và một file PNG hợp lệ sẽ bị báo là "không rõ định dạng".
	_ "image/gif"
	_ "image/png"

	"golang.org/x/image/draw"
)

const (
	// MaxBytes caps what the server will read from one upload.
	// MaxBytes giới hạn số byte server đọc từ một lần tải lên.
	MaxBytes = 5 << 20 // 5 MB

	// Size is the side of the square the avatar is stored at. Anything larger
	// is bandwidth spent on pixels no part of the interface shows.
	//
	// Size là cạnh của hình vuông mà avatar được lưu. Lớn hơn nữa chỉ là
	// băng thông tiêu cho những điểm ảnh mà không chỗ nào trong giao diện
	// hiển thị tới.
	Size = 256

	// maxDimension is checked from the header before the pixels are decoded.
	// A small file can declare enormous dimensions, and decoding it would
	// allocate width*height*4 bytes — a few hundred kilobytes on the wire
	// becoming gigabytes of memory.
	//
	// maxDimension được kiểm từ phần header trước khi giải mã điểm ảnh. Một
	// file nhỏ vẫn có thể khai báo kích thước khổng lồ, và giải mã nó sẽ cấp
	// phát width*height*4 byte — vài trăm kilobyte truyền qua mạng biến thành
	// hàng gigabyte bộ nhớ.
	maxDimension = 8000
)

var (
	ErrTooLarge   = errors.New("the image is too large")
	ErrNotAnImage = errors.New("the file is not a JPEG, PNG or GIF image")
)

// Avatar reads an uploaded file and returns a square JPEG of Size by Size.
//
// It decodes and re-encodes rather than storing the original bytes, which buys
// three things at once. Metadata is dropped, and a photo taken on a phone
// routinely carries the GPS coordinates of where it was taken — publishing
// that with someone's avatar would hand out their home address. The output is
// known to be a real image rather than a file that is both a valid image and a
// valid script, which is how an upload form becomes a way to serve active
// content from this origin. And the stored file is bounded, so one 5 MB photo
// does not get sent to every viewer of a feed.
//
// Avatar đọc file được tải lên và trả về một ảnh JPEG vuông Size x Size.
//
// Nó giải mã rồi mã hoá lại thay vì lưu nguyên byte gốc, và điều đó đem lại ba
// thứ cùng lúc. Metadata bị loại bỏ — ảnh chụp bằng điện thoại thường mang
// theo toạ độ GPS nơi chụp, công bố nó kèm avatar đồng nghĩa với việc phát đi
// địa chỉ nhà của người ta. Kết quả chắc chắn là ảnh thật chứ không phải một
// file vừa hợp lệ như ảnh vừa hợp lệ như mã script, vốn là cách một form tải
// lên biến thành đường phục vụ nội dung thực thi từ chính origin này. Và file
// lưu xuống có kích thước giới hạn, nên một tấm ảnh 5 MB không bị gửi tới mọi
// người xem feed.
func Avatar(r io.Reader) ([]byte, error) {
	// One byte past the limit is enough to tell "exactly at the limit" from
	// "over it", without reading the rest of an oversized upload.
	//
	// Đọc dư một byte là đủ để phân biệt "vừa đúng giới hạn" với "vượt quá",
	// mà không phải đọc nốt phần còn lại của một file quá khổ.
	raw, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxBytes {
		return nil, ErrTooLarge
	}
	if len(raw) == 0 {
		return nil, ErrNotAnImage
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrNotAnImage
	}
	if config.Width > maxDimension || config.Height > maxDimension {
		return nil, ErrTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrNotAnImage
	}

	out := image.NewRGBA(image.Rect(0, 0, Size, Size))
	draw.CatmullRom.Scale(out, out.Bounds(), src, squareOf(src.Bounds()), draw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode avatar: %w", err)
	}
	return buf.Bytes(), nil
}

// squareOf returns the largest centred square inside bounds.
//
// Scaling a rectangle straight into a square would stretch the picture; taking
// the middle square first keeps faces in proportion, and the middle is where
// the subject of a portrait almost always is.
//
// squareOf trả về hình vuông lớn nhất nằm giữa bounds.
//
// Co thẳng một hình chữ nhật vào khung vuông sẽ làm ảnh bị kéo méo; cắt lấy
// phần vuông ở giữa trước thì khuôn mặt giữ đúng tỉ lệ, và phần giữa gần như
// luôn là nơi có chủ thể của một tấm ảnh chân dung.
func squareOf(bounds image.Rectangle) image.Rectangle {
	side := min(bounds.Dx(), bounds.Dy())
	x := bounds.Min.X + (bounds.Dx()-side)/2
	y := bounds.Min.Y + (bounds.Dy()-side)/2
	return image.Rect(x, y, x+side, y+side)
}
