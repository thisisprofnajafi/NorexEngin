package handler

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"norex/database"
	"norex/models"
)

func CreateRole(c *fiber.Ctx) error {
	var role models.Role
	if err := c.BodyParser(&role); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	role.ID = primitive.NewObjectID() // Automatically generate ObjectID

	collection := database.GetCollection("roles")
	_, err := collection.InsertOne(context.TODO(), role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create role"})
	}

	return c.JSON(fiber.Map{"message": "Role created successfully", "role": role})
}

func GetRole(c *fiber.Ctx) error {
	roleID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
	}

	collection := database.GetCollection("roles")
	var role models.Role
	err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&role)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Role not found"})
	}

	return c.JSON(role)
}

func UpdateRole(c *fiber.Ctx) error {
	roleID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
	}

	var updatedRole models.Role
	if err := c.BodyParser(&updatedRole); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	collection := database.GetCollection("roles")
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": objectID}, bson.M{"$set": updatedRole})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update role"})
	}

	return c.JSON(fiber.Map{"message": "Role updated successfully"})
}

func DeleteRole(c *fiber.Ctx) error {
	roleID := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
	}

	collection := database.GetCollection("roles")
	_, err = collection.DeleteOne(context.TODO(), bson.M{"_id": objectID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete role"})
	}

	return c.JSON(fiber.Map{"message": "Role deleted successfully"})
}

func ListRoles(c *fiber.Ctx) error {
	collection := database.GetCollection("roles")
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch roles"})
	}
	defer cursor.Close(context.TODO())

	var roles []models.Role
	if err = cursor.All(context.TODO(), &roles); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to parse roles"})
	}

	return c.JSON(roles)
}
