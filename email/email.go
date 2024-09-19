package email

import (
	"fmt"
	"github.com/go-mail/mail"
)

const (
	smtpHost     = "smtp.gmail.com"
	smtpPort     = 587
	smtpUsername = "thisisprofnajafi@gmail.com"
	smtpPassword = "jstt whdr yluf uccz"
	fromEmail    = "thisisprofnajafi@gmail.com"
)

func SendVerificationEmail(toEmail, subject, body string) error {
	m := mail.NewMessage()

	// Set the sender email
	m.SetHeader("From", fromEmail)

	// Set the recipient email (dynamic)
	m.SetHeader("To", toEmail)

	// Set the email subject (dynamic)
	m.SetHeader("Subject", subject)

	// Set the email body (dynamic)
	m.SetBody("text/html", GenerateVerificationEmailBody(body))

	// Create a new dialer with SMTP credentials
	d := mail.NewDialer(smtpHost, smtpPort, smtpUsername, smtpPassword)

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		fmt.Printf("Failed to send email: %v\n", err) // Log the actual error
		return err
	}

	return nil
}
