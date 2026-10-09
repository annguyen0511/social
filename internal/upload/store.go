package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// Store keeps avatar files in one directory on disk.
//
// Disk is the right amount of machinery for a single server. It is also the
// wrong answer for more than one: two instances behind a load balancer would
// each hold half the avatars and answer 404 for the other half, and a
// container restart loses everything unless the directory is a mounted volume.
// Moving to object storage later means replacing this type, and nothing else.
//
// Store giữ các file avatar trong một thư mục trên đĩa.
//
// Lưu xuống đĩa là mức vừa đủ cho một server duy nhất. Nó cũng là câu trả lời
// sai cho nhiều hơn một: hai tiến trình sau một bộ cân bằng tải sẽ mỗi bên giữ
// một nửa số avatar và trả 404 cho nửa còn lại, còn khởi động lại container là
// mất sạch nếu thư mục không phải volume được gắn vào. Sau này chuyển sang
// object storage thì chỉ phải thay type này, không đụng gì khác.
type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir %q: %w", dir, err)
	}
	return &Store{dir: dir}, nil
}

// Save writes data under a name this server chooses.
//
// The name never comes from the client. A browser sends whatever the file was
// called, and "../../etc/passwd" is a perfectly ordinary string to put there —
// joining it onto a directory is how an upload form becomes a way to write
// anywhere on the disk. Random names also mean one avatar's address gives away
// nothing about anyone else's.
//
// Save ghi data dưới một cái tên do chính server đặt.
//
// Tên không bao giờ đến từ client. Trình duyệt gửi đúng tên file vốn có, mà
// "../../etc/passwd" là một chuỗi hoàn toàn bình thường để đặt vào đó — ghép
// nó vào một thư mục chính là cách một form tải lên biến thành đường ghi file
// vào bất cứ đâu trên đĩa. Tên ngẫu nhiên còn khiến địa chỉ của một avatar
// không tiết lộ gì về avatar của người khác.
func (s *Store) Save(data []byte) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	name := hex.EncodeToString(raw) + ".jpg"

	if err := os.WriteFile(filepath.Join(s.dir, name), data, 0o644); err != nil {
		return "", fmt.Errorf("write avatar: %w", err)
	}
	return name, nil
}

// Remove deletes the file an avatar URL points at. A URL that does not look
// like one of ours, or a file already gone, is not an error: the caller only
// wants the old picture to stop existing.
//
// Remove xoá file mà một avatar URL trỏ tới. URL không giống của mình, hoặc
// file vốn đã biến mất, đều không phải lỗi: phía gọi chỉ muốn ảnh cũ không còn
// tồn tại nữa.
func (s *Store) Remove(avatarURL string) error {
	name := path.Base(avatarURL)
	if !s.valid(name) {
		return nil
	}

	err := os.Remove(filepath.Join(s.dir, name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Path resolves a requested name to a file, or returns false. Callers serve
// whatever comes back, so this is the gate: only the exact shape Save produces
// gets through, which leaves no room for a name that walks out of the
// directory.
//
// Path chuyển tên được yêu cầu thành đường dẫn file, hoặc trả về false. Phía
// gọi sẽ phục vụ bất cứ thứ gì nhận được, nên đây chính là cổng chặn: chỉ đúng
// hình dạng mà Save tạo ra mới lọt qua, nên không còn chỗ cho một cái tên đi
// ngược ra khỏi thư mục.
func (s *Store) Path(name string) (string, bool) {
	if !s.valid(name) {
		return "", false
	}
	return filepath.Join(s.dir, name), true
}

// valid accepts only "<32 hex characters>.jpg".
// valid chỉ chấp nhận "<32 ký tự hex>.jpg".
func (s *Store) valid(name string) bool {
	const suffix = ".jpg"
	if len(name) != 32+len(suffix) || name[32:] != suffix {
		return false
	}
	_, err := hex.DecodeString(name[:32])
	return err == nil
}
