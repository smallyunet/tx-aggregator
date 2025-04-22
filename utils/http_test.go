package utils

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ------------------------
// Test for GetInsensitiveQuery
// ------------------------
func TestGetInsensitiveQuery(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		val1 := GetInsensitiveQuery(c, "foo")
		val2 := GetInsensitiveQuery(c, "FOO")
		val3 := GetInsensitiveQuery(c, "FoO")
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

// ------------------------
// Test for DoHttpRequestWithLogging
// ------------------------
func TestDoHttpRequestWithLogging(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		defer r.Body.Close()

		var data map[string]string
		err = json.Unmarshal(body, &data)
		assert.NoError(t, err)
		assert.Equal(t, "test", data["name"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Define request payload and result holder
	payload := map[string]string{"name": "test"}
	var result map[string]string

	err := DoHttpRequestWithLogging("POST", "test_label", server.URL, payload, map[string]string{
		"Content-Type": "application/json",
	}, &result)

	assert.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

// ------------------------
// Test non-200 response case
// ------------------------
func TestDoHttpRequestWithLogging_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	err := DoHttpRequestWithLogging("GET", "test_400", server.URL, nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-200 response")
}

// ------------------------
// Test malformed response body
// ------------------------
func TestDoHttpRequestWithLogging_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid_json}`))
	}))
	defer server.Close()

	var result map[string]string
	err := DoHttpRequestWithLogging("GET", "bad_json", server.URL, nil, nil, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal response failed")
}

// ------------------------
// Test request marshal failure
// ------------------------
func TestDoHttpRequestWithLogging_BadMarshal(t *testing.T) {
	ch := make(chan int) // non-marshalable type
	err := DoHttpRequestWithLogging("POST", "bad_marshal", "http://example.com", ch, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal request failed")
}
