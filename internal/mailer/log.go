package mailer

// LogClient writes the activation link to the log instead of mailing it. It
// keeps registration usable on a machine with no SendGrid key; main refuses to
// fall back to it in production, where silently sending nothing would leave
// every new account stranded.
//
// LogClient ghi đường dẫn kích hoạt ra log thay vì gửi mail. Nó giữ cho luồng
// đăng ký dùng được trên máy chưa có key SendGrid; main từ chối dùng bản này ở
// production, vì im lặng không gửi gì sẽ khiến mọi tài khoản mới bị kẹt.
type LogClient struct {
	logw func(msg string, keysAndValues ...any)
}

// NewLog takes a structured log function, so a zap SugaredLogger's Infow can
// be passed straight in.
//
// NewLog nhận một hàm log có cấu trúc, nên truyền thẳng Infow của zap
// SugaredLogger vào được.
func NewLog(logw func(msg string, keysAndValues ...any)) *LogClient {
	return &LogClient{logw: logw}
}

func (c *LogClient) SendActivation(to, username, activationURL string) error {
	// Rendering here too, so a broken template is caught in development
	// rather than the first time production sends a real message.
	//
	// Vẫn render ở đây, để template hỏng bị phát hiện ngay lúc dev chứ không
	// phải tới lần đầu production gửi thư thật.
	subject, _, _, err := renderInvitation(username, activationURL)
	if err != nil {
		return err
	}

	c.logw("mailer disabled, activation link not sent",
		"to", to, "username", username, "subject", subject, "activation_url", activationURL)
	return nil
}
