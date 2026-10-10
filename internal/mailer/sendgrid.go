package mailer

import (
	"fmt"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

const (
	maxSendRetries = 3
	retryBackoff   = time.Second
)

// SendGridClient sends mail through SendGrid's v3 API.
// SendGridClient gửi mail qua API v3 của SendGrid.
type SendGridClient struct {
	fromEmail string
	fromName  string
	sandbox   bool
	client    *sendgrid.Client
}

// NewSendGrid builds a client. With sandbox true, SendGrid validates every
// request and returns 200 but delivers nothing — the way to exercise this code
// path without mailing real people.
//
// NewSendGrid tạo client. Khi sandbox bằng true, SendGrid vẫn kiểm tra toàn bộ
// request và trả 200 nhưng không gửi đi đâu cả — cách để chạy thử luồng này mà
// không làm phiền người thật.
func NewSendGrid(apiKey, fromEmail, fromName string, sandbox bool) *SendGridClient {
	return &SendGridClient{
		fromEmail: fromEmail,
		fromName:  fromName,
		sandbox:   sandbox,
		client:    sendgrid.NewSendClient(apiKey),
	}
}

func (c *SendGridClient) SendActivation(to, username, activationURL string) error {
	return c.send(invitationTemplate, to, username, activationURL)
}

func (c *SendGridClient) SendPasswordReset(to, username, resetURL string) error {
	return c.send(passwordResetTemplate, to, username, resetURL)
}

func (c *SendGridClient) send(path, to, username, actionURL string) error {
	subject, plain, html, err := render(path, username, actionURL)
	if err != nil {
		return err
	}

	message := mail.NewSingleEmail(
		mail.NewEmail(c.fromName, c.fromEmail),
		subject,
		mail.NewEmail(username, to),
		plain,
		html,
	)

	if c.sandbox {
		settings := mail.NewMailSettings()
		settings.SetSandboxMode(mail.NewSetting(true))
		message.SetMailSettings(settings)
	}

	// SendGrid occasionally answers 429 or 5xx under load. Retrying a few
	// times costs little and turns a transient blip into a delivered mail;
	// a rejected message (4xx other than 429) is returned straight away
	// because retrying it would fail identically.
	//
	// SendGrid thỉnh thoảng trả 429 hoặc 5xx khi tải cao. Thử lại vài lần tốn
	// rất ít mà biến một trục trặc nhất thời thành email gửi được; còn thư bị
	// từ chối hẳn (4xx không phải 429) thì trả lỗi ngay, vì thử lại cũng hỏng
	// y như vậy.
	var lastErr error
	for attempt := 1; attempt <= maxSendRetries; attempt++ {
		resp, err := c.client.Send(message)
		if err != nil {
			lastErr = err
		} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		} else {
			lastErr = fmt.Errorf("sendgrid responded %d: %s", resp.StatusCode, resp.Body)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
				return lastErr
			}
		}

		if attempt < maxSendRetries {
			time.Sleep(retryBackoff * time.Duration(attempt))
		}
	}

	return fmt.Errorf("send mail to %s after %d attempts: %w", to, maxSendRetries, lastErr)
}
