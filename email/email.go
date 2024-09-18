package email

import (
	"net/smtp"
	"strconv"
)

const (
	smtpHost     = "smtp.gmail.com"
	smtpPort     = 587
	smtpUsername = "thisisprofnajafi@gmail.com"
	smtpPassword = "jstt whdr yluf uccz"
	fromEmail    = "thisisprofnajafi@gmail.com"
)

func SendEmail(toEmail, subject, body string) error {
	auth := smtp.PlainAuth(
		"",
		smtpUsername,
		smtpPassword,
		smtpHost,
	)

	msg := "Subject: " + subject + "\r\n" + body

	err := smtp.SendMail(
		smtpHost+":"+strconv.Itoa(smtpPort),
		auth,
		fromEmail,
		[]string{toEmail},
		[]byte(msg),
	)

	if err != nil {
		return err
	}

	return nil
}
