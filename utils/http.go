package utils

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GetInsensitiveQuery retrieves the query parameter by ignoring case sensitivity.
func GetInsensitiveQuery(ctx *fiber.Ctx, key string) string {
	for k, v := range ctx.Queries() {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}
