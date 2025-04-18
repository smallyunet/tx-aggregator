package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetInsensitiveQuery(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		// lowercase key
		val1 := GetInsensitiveQuery(c, "foo")
		// uppercase key
		val2 := GetInsensitiveQuery(c, "FOO")
		// mixed-case key
		val3 := GetInsensitiveQuery(c, "FoO")
		// non-existent key
		val4 := GetInsensitiveQuery(c, "bar")

		assert.Equal(t, "123", val1)
		assert.Equal(t, "123", val2)
		assert.Equal(t, "123", val3)
		assert.Equal(t, "", val4)

		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test?FoO=123", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
