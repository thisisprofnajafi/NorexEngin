package email

import (
	"fmt"
)

func GenerateVerificationEmailBody(code string) string {
	return fmt.Sprintf(`
        <html>
        <body>
            <h1>Your Verification Code</h1>
            <p>Your verification code is: <strong>%s</strong></p>
            <p>This code will expire in 10 minutes.</p>
        </body>
        </html>
    `, code)
}
