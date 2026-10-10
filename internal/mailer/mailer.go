package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

const (
	invitationTemplate    = "templates/user_invitation.tmpl"
	passwordResetTemplate = "templates/password_reset.tmpl"
)

// FS holds the mail templates compiled into the binary, so a deployment is a
// single file with no template directory to ship alongside it.
//
// FS chứa các template email được nhúng thẳng vào binary, nên khi triển khai
// chỉ cần một file duy nhất, không phải mang theo thư mục template.
//
//go:embed templates
var FS embed.FS

// Client sends the transactional mail the API needs. The interface exists so
// that main can pick a real provider or a logging stand-in, and so tests can
// substitute a fake without touching the network.
//
// Client gửi các email giao dịch mà API cần. Có interface để main chọn được
// nhà cung cấp thật hay bản ghi log thay thế, và để test thay bằng bản giả mà
// không đụng tới mạng.
type Client interface {
	// SendActivation mails username an invitation link. It returns an error
	// only when the message could not be handed over to the provider.
	//
	// SendActivation gửi cho username đường dẫn kích hoạt. Chỉ trả lỗi khi
	// không bàn giao được thư cho nhà cung cấp.
	SendActivation(to, username, activationURL string) error

	// SendPasswordReset mails username a link for choosing a new password.
	// Same contract: an error only when the provider would not take it.
	//
	// SendPasswordReset gửi cho username đường dẫn để chọn mật khẩu mới.
	// Cùng giao kèo: chỉ trả lỗi khi nhà cung cấp không nhận thư.
	SendPasswordReset(to, username, resetURL string) error
}

// mailData is what the templates render against. Both mails carry the same
// two things — who it is for and the one link it exists to deliver — so they
// share a shape rather than each having a field named after its own purpose.
//
// mailData là dữ liệu mà template dùng để render. Cả hai loại thư đều mang
// đúng hai thứ — gửi cho ai và một đường dẫn duy nhất mà nó sinh ra để
// chuyển đi — nên chúng dùng chung một hình dạng, thay vì mỗi loại có một
// field đặt tên theo mục đích riêng.
type mailData struct {
	Username  string
	ActionURL string
}

// render produces the subject, the plain text body and the HTML body from one
// template file. Mail clients that cannot show HTML fall back to the plain
// part, and some spam filters score a message lower when it is missing.
//
// render tạo ra tiêu đề, phần thân dạng chữ thuần và phần thân HTML từ cùng
// một file template. Trình đọc mail không hiển thị được HTML sẽ dùng phần chữ
// thuần, và một số bộ lọc spam chấm điểm thấp nếu thiếu phần này.
func render(path, username, actionURL string) (subject, plain, html string, err error) {
	tmpl, err := template.ParseFS(FS, path)
	if err != nil {
		return "", "", "", fmt.Errorf("parse %s: %w", path, err)
	}

	data := mailData{Username: username, ActionURL: actionURL}

	parts := make(map[string]string, 3)
	for _, name := range []string{"subject", "plainBody", "htmlBody"} {
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
			return "", "", "", fmt.Errorf("render %s: %w", name, err)
		}
		parts[name] = buf.String()
	}

	return parts["subject"], parts["plainBody"], parts["htmlBody"], nil
}
