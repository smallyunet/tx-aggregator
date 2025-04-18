package router

import (
	"github.com/gofiber/fiber/v2"
	"tx-aggregator/api"
)

// SetupRoutes configures all HTTP routes and associates them with their respective handlers.
// Parameters:
//   - app: Fiber application instance
//   - txHandler: TransactionHandler to process transaction-related endpoints
func SetupRoutes(app *fiber.App, txHandler *api.TransactionHandler) {
	// Health check endpoint (useful for Docker, Kubernetes, load balancers, etc.)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Transaction APIs
	app.Get("/transactions", txHandler.GetTransactions)
}
