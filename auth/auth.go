package auth

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"math/rand"
	"norex/database"
	"norex/email"
	"norex/models"
	"time"
)

func generateVerificationCode() string {
	code := make([]byte, 5)
	rand.Read(code)
	return fmt.Sprintf("%05d", rand.Intn(100000))
}

func sendEmail(emailAddr, code, subject string) error {
	// Use the new GenerateVerificationEmailBody function for the email body
	body := email.GenerateVerificationEmailBody(code)
	return email.SendEmail(emailAddr, subject, body)
}
func RequestCode(c *fiber.Ctx) error {
	emailAddress := c.FormValue("email")

	// Check if user exists, if not, create a new user
	collection := database.GetCollection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": emailAddress}).Decode(&user)

	if err != nil {
		// New user, generate a new record
		user = models.User{
			Email:            emailAddress,
			VerificationCode: generateVerificationCode(),
			CodeExpiryTime:   time.Now().Add(10 * time.Minute),
			AttemptCount:     0,
		}
		_, err := collection.InsertOne(context.TODO(), user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user"})
		}
	} else {
		// Existing user, update the code and expiry
		user.VerificationCode = generateVerificationCode()
		user.CodeExpiryTime = time.Now().Add(10 * time.Minute)
		user.AttemptCount = 0 // Reset attempts
		_, err := collection.UpdateOne(context.TODO(), bson.M{"email": emailAddress}, bson.M{"$set": user})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update code"})
		}
	}

	// Create an HTML email body
	htmlBody := fmt.Sprintf(`
        <html>
        <body>
            <h2>Hello,</h2>
            <p>Your verification code is: <strong>%s</strong></p>
            <p>Please use this code to verify your email. This code will expire in 10 minutes.</p>
            <br>
            <p>Best regards,</p>
            <p>Your Company Team</p>
        </body>
        </html>
    `, user.VerificationCode)

	// Use the email package's SendEmail function to send the HTML email
	err = email.SendEmail(user.Email, "Verify Email", htmlBody)
	if err != nil {
		// Capture the error details and log them
		fmt.Printf("Error occurred while sending email to %s: %v\n", user.Email, err)

		// If the error has an underlying cause, log that too
		if netErr, ok := err.(interface{ Unwrap() error }); ok && netErr.Unwrap() != nil {
			fmt.Printf("Underlying error: %v\n", netErr.Unwrap())
		}

		// Return the error message in the response
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to send email: %v", err),
		})
	}

	return c.JSON(fiber.Map{"message": "Verification code sent"})
}
func VerifyCode(c *fiber.Ctx) error {
	email := c.FormValue("email")
	code := c.FormValue("code")

	collection := database.GetCollection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email"})
	}

	// Check if user is banned
	if time.Now().Before(user.BanUntil) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "User is banned until " + user.BanUntil.String()})
	}

	// Check if the code is expired
	if time.Now().After(user.CodeExpiryTime) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Code expired, request a new one"})
	}

	// Verify the code
	if code != user.VerificationCode {
		user.AttemptCount++
		if user.AttemptCount >= 5 {
			user.BanUntil = time.Now().Add(2 * time.Hour) // Ban for 2 hours
		}

		_, _ = collection.UpdateOne(context.TODO(), bson.M{"email": email}, bson.M{"$set": user})
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid code"})
	}

	// Reset attempts on success and generate token
	user.AttemptCount = 0
	_, _ = collection.UpdateOne(context.TODO(), bson.M{"email": email}, bson.M{"$set": user})

	// Generate JWT token
	token, err := generateToken(user.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{"token": token})
}

func updateBanStatus(user *models.User) {
	if user.AttemptCount >= 5 {
		if time.Now().After(user.BanUntil.Add(2 * time.Hour)) {
			user.BanUntil = time.Now().Add(24 * time.Hour) // Ban for 24 hours on second failure
		}
	}
}

var jwtSecret = []byte("your_jwt_secret_key")

func generateToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 72).Unix(), // 72 hours expiry
	})

	return token.SignedString(jwtSecret)
}
