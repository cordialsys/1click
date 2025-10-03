package endpoints

import (
	"github.com/cordialsys/panel/pkg/admin"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

func (endpoints *Endpoints) AdminUsers(c *fiber.Ctx) error {
	if endpoints.panel.ApiKey == "" {
		return servererrors.FailedPreconditionf("not activated")
	}
	client := admin.NewClient(endpoints.panel.ApiKey)
	nextPageToken := c.Query("page_token")
	usersPage, err := client.ListUsers(nextPageToken)
	if err != nil {
		return servererrors.BadRequestf("failed to get users: %v", err)
	}
	return c.JSON(usersPage)
}
