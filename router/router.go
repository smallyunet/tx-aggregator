package router

import (
	"github.com/gofiber/fiber/v2"
	"tx-aggregator/api"
)

// SetupRoutes binds all the routes to the app
func SetupRoutes(app *fiber.App) {
	app.Get("/transactions", api.GetTransactions)
}
