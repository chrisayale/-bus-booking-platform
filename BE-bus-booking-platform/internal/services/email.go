package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
)

type EmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	appURL   string
}

func NewEmailService(host string, port int, username, password, from, appURL string) *EmailService {
	return &EmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		appURL:   appURL,
	}
}

func (s *EmailService) Send(to, subject, htmlBody string) error {
	if s.host == "" || s.port == 0 {
		return fmt.Errorf("SMTP not configured")
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, htmlBody)

	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}

func (s *EmailService) SendVerificationEmail(to, name, token string) error {
	link := fmt.Sprintf("%s/auth/verify?token=%s", s.appURL, token)
	subject := "Verify your BusBook account"

	body, _ := renderTemplate(emailVerificationHTML, map[string]string{
		"Name":   name,
		"Link":   link,
		"AppURL": s.appURL,
	})

	return s.Send(to, subject, body)
}

func (s *EmailService) SendPasswordResetEmail(to, name, token string) error {
	link := fmt.Sprintf("%s/auth/reset-password?token=%s", s.appURL, token)
	subject := "Reset your BusBook password"

	body, _ := renderTemplate(passwordResetHTML, map[string]string{
		"Name":   name,
		"Link":   link,
		"AppURL": s.appURL,
	})

	return s.Send(to, subject, body)
}

func (s *EmailService) SendTicketInvoice(to, name, reference string, tickets []TicketInfo, downloadLink string) error {
	subject := fmt.Sprintf("Your BusBook Tickets - %s", reference)

	body, _ := renderTemplate(ticketInvoiceHTML, map[string]interface{}{
		"Name":         name,
		"Reference":    reference,
		"Tickets":      tickets,
		"DownloadLink": downloadLink,
		"AppURL":       s.appURL,
	})

	return s.Send(to, subject, body)
}

type TicketInfo struct {
	PassengerName string
	SeatNumber    string
	Date          string
	Time          string
	Origin        string
	Destination   string
	Price         string
}

func renderTemplate(tmpl string, data interface{}) (string, error) {
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ==================== HTML TEMPLATES ====================

const emailVerificationHTML = `
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; background: #f4f4f4; padding: 40px;">
  <div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; padding: 32px;">
    <h2 style="color: #2563eb;">Welcome to BusBook!</h2>
    <p>Hi {{.Name}},</p>
    <p>Thanks for signing up! Please verify your email address by clicking the button below:</p>
    <a href="{{.Link}}" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; font-weight: bold; margin: 16px 0;">
      Verify Email
    </a>
    <p style="color: #666; font-size: 14px;">Or copy this link: {{.Link}}</p>
    <p style="color: #666; font-size: 14px;">This link expires in 24 hours.</p>
    <hr style="border: none; border-top: 1px solid #eee; margin: 24px 0;">
    <p style="color: #999; font-size: 12px;">BusBook &mdash; {{.AppURL}}</p>
  </div>
</body>
</html>`

const passwordResetHTML = `
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; background: #f4f4f4; padding: 40px;">
  <div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; padding: 32px;">
    <h2 style="color: #2563eb;">Reset Your Password</h2>
    <p>Hi {{.Name}},</p>
    <p>We received a request to reset your password. Click the button below to create a new one:</p>
    <a href="{{.Link}}" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; font-weight: bold; margin: 16px 0;">
      Reset Password
    </a>
    <p style="color: #666; font-size: 14px;">Or copy this link: {{.Link}}</p>
    <p style="color: #666; font-size: 14px;">This link expires in 1 hour. If you didn't request this, ignore this email.</p>
    <hr style="border: none; border-top: 1px solid #eee; margin: 24px 0;">
    <p style="color: #999; font-size: 12px;">BusBook &mdash; {{.AppURL}}</p>
  </div>
</body>
</html>`

const ticketInvoiceHTML = `
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: Arial, sans-serif; background: #f4f4f4; padding: 40px;">
  <div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; padding: 32px;">
    <h2 style="color: #2563eb;">Your Tickets Are Ready!</h2>
    <p>Hi {{.Name}},</p>
    <p>Your booking <strong>{{.Reference}}</strong> is confirmed.</p>

    <table style="width: 100%; border-collapse: collapse; margin: 16px 0;">
      <tr style="background: #f8fafc;">
        <th style="padding: 8px 12px; text-align: left; border-bottom: 1px solid #e2e8f0;">Passenger</th>
        <th style="padding: 8px 12px; text-align: left; border-bottom: 1px solid #e2e8f0;">Seat</th>
        <th style="padding: 8px 12px; text-align: left; border-bottom: 1px solid #e2e8f0;">Price</th>
      </tr>
      {{range .Tickets}}
      <tr>
        <td style="padding: 8px 12px; border-bottom: 1px solid #f1f5f9;">{{.PassengerName}}</td>
        <td style="padding: 8px 12px; border-bottom: 1px solid #f1f5f9;">{{.SeatNumber}}</td>
        <td style="padding: 8px 12px; border-bottom: 1px solid #f1f5f9;">{{.Price}} MAD</td>
      </tr>
      {{end}}
    </table>

    <a href="{{.DownloadLink}}" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; font-weight: bold; margin: 16px 0;">
      Download Tickets
    </a>
    <p style="color: #666; font-size: 13px;">This download link can only be used once and expires in 7 days.</p>
    <hr style="border: none; border-top: 1px solid #eee; margin: 24px 0;">
    <p style="color: #999; font-size: 12px;">BusBook &mdash; {{.AppURL}}</p>
  </div>
</body>
</html>`
