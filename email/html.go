package email

import (
	"fmt"
)

func GenerateVerificationEmailBody(code string) string {
	return fmt.Sprintf(`
        <html>
        <head>
            <style>
                body {
                    font-family: Arial, sans-serif;
                    color: #191A19;
                    background-color: #D8E9A8;
                    margin: 0;
                    padding: 20px;
                }
                h1 {
                    color: #1E5128;
                }
                p {
                    color: #4E9F3D;
                }
                .footer {
                    margin-top: 20px;
                    border-top: 1px solid #191A19;
                    padding-top: 10px;
                    text-align: center;
                }
                .footer a {
                    color: #1E5128;
                    text-decoration: none;
                }
            </style>
        </head>
        <body>
            <h1>Your Verification Code</h1>
            <p>Your verification code is: <strong>%s</strong></p>
            <p>This code will expire in 5 minutes.</p>
            <div class="footer">
                <p>&copy; 2024 Norex. All rights reserved.</p>
                <p>Visit us at <a href="https://norex.app" target="_blank">norex.app</a></p>
            </div>
        </body>
        </html>
    `, code)
}
