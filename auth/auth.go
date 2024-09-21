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
	"strings"
	"time"
)

func generateVerificationCode() string {
	code := make([]byte, 5)
	rand.Read(code)
	return fmt.Sprintf("%05d", rand.Intn(100000))
}

func generateUniqueID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 7)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func RequestCode(c *fiber.Ctx) error {
	fmt.Printf("Request Body: %s\n", c.Body())

	var body struct {
		Email string `json:"email"`
	}

	// Parse the JSON body using BodyParser, which will use your custom decoder
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	emailAddress := body.Email
	fmt.Println(emailAddress)
	// Check if user exists, if not, create a new user
	collection := database.GetCollection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"email": emailAddress}).Decode(&user)

	if err != nil {
		// New user, generate a new record
		user = models.User{
			Email:            emailAddress,
			VerificationCode: generateVerificationCode(),
			UniqueID:         generateUniqueID(),
			CodeExpiryTime:   time.Now().UTC().Add(5 * time.Minute),
			AttemptCount:     0,
		}
		_, err := collection.InsertOne(context.TODO(), user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user"})
		}
	} else {
		// Existing user, check if they are allowed to request a new code
		lastRequestTime := user.CodeExpiryTime // Assuming CodeExpiryTime is used for this purpose
		if time.Since(lastRequestTime) < 5*time.Minute {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "You can request a new code every 5 minutes",
			})
		}

		// Update the code and expiry
		user.VerificationCode = generateVerificationCode()
		user.CodeExpiryTime = time.Now().UTC().Add(5 * time.Minute)
		user.AttemptCount = 0 // Reset attempts
		_, err := collection.UpdateOne(context.TODO(), bson.M{"email": emailAddress}, bson.M{"$set": user})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update code"})
		}
	}

	// Use the email package's SendEmail function to send the HTML email
	err = email.SendVerificationEmail(user.Email, "Verify Email", user.VerificationCode)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to send email: %v", err),
		})
	}

	return c.JSON(fiber.Map{"message": "Verification code sent"})
}

func VerifyCode(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}

	// Parse the JSON body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userEmail := body.Email
	code := body.Code

	collection := database.GetCollection("users")
	var user models.User

	// Find the user by email
	err := collection.FindOne(context.TODO(), bson.M{"email": userEmail}).Decode(&user)
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
		_, _ = collection.UpdateOne(context.TODO(), bson.M{"email": userEmail}, bson.M{"$set": user})
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid code"})
	}

	// Reset attempts on success
	user.AttemptCount = 0

	// Assign a default role if the user doesn't have one
	if user.Role == "" {
		user.Role = "user"
		user.VerifiedEmailDate = time.Now().UTC() // Assign "user" as the default role
	}

	// Update user in the database with reset attempts and role if needed
	_, _ = collection.UpdateOne(context.TODO(), bson.M{"email": userEmail}, bson.M{"$set": user})

	// Generate JWT token
	token, err := generateToken(user.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	// Fetch the user's role from the database
	roleCollection := database.GetCollection("roles")
	var role models.Role
	err = roleCollection.FindOne(context.TODO(), bson.M{"name": user.Role}).Decode(&role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve user role"})
	}

	// Create a session for the user
	err = CreateSession(user, role, token, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create session"})
	}

	// Determine if the user needs to provide additional information
	needInfo := user.Name == "" || user.Gender == ""

	return c.JSON(fiber.Map{
		"message":      "Verification successful",
		"require_info": needInfo,
		"token":        token,
	})
}

func updateBanStatus(user *models.User) {
	if user.AttemptCount >= 5 {
		if time.Now().After(user.BanUntil.Add(2 * time.Hour)) {
			user.BanUntil = time.Now().Add(24 * time.Hour) // Ban for 24 hours on second failure
		}
	}
}

var JWTSecret = []byte("your_jwt_secret_key")

func generateToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 72).Unix(), // 72 hours expiry
	})

	return token.SignedString(JWTSecret)
}

func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or invalid token"})
		}

		// Remove "Bearer " prefix if present
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
		}

		email, ok := claims["email"].(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email not found in token"})
		}

		// Set email in Locals
		c.Locals("email", email)

		return c.Next()
	}
}

func CreateSession(user models.User, role models.Role, token string, ipAddress string, device string) error {
	collection := database.GetCollection("sessions")

	session := models.Session{
		UserID:    user.ID,
		Token:     token,
		IPAddress: ipAddress,
		Device:    device,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(role.SessionExpiry), // Expiry based on role
		Role:      role.Name,
	}

	_, err := collection.InsertOne(context.TODO(), session)
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	return nil
}
