package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

const invitationTemplate = "templates/user_invitation.tmpl"

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
}

// invitationData is what the templates render against.
// invitationData là dữ liệu mà template dùng để render.
type invitationData struct {
	Username      string
	ActivationURL string
}

// renderInvitation produces the subject, the plain text body and the HTML body
// from one template file. Mail clients that cannot show HTML fall back to the
// plain part, and some spam filters score a message lower when it is missing.
//
// renderInvitation tạo ra tiêu đề, phần thân dạng chữ thuần và phần thân HTML
// từ cùng một file template. Trình đọc mail không hiển thị được HTML sẽ dùng
// phần chữ thuần, và một số bộ lọc spam chấm điểm thấp nếu thiếu phần này.
func renderInvitation(username, activationURL string) (subject, plain, html string, err error) {
	tmpl, err := template.ParseFS(FS, invitationTemplate)
	if err != nil {
		return "", "", "", fmt.Errorf("parse %s: %w", invitationTemplate, err)
	}

	data := invitationData{Username: username, ActivationURL: activationURL}

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
