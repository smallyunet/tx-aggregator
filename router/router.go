package router

import (
	"tx-aggregator/api"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes binds all the routes to the app
func SetupRoutes(app *fiber.App) {
	app.Get("/transactions", api.GetTransactions)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
}
