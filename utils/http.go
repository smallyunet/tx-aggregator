package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"tx-aggregator/logger"

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

// DoHttpRequestWithLogging performs an HTTP request with optional JSON body and optional JSON decoding of the response.
// It logs request method, URL, duration, response size, status, and error if any.
//
// method:     "GET", "POST", etc.
// url:        full request URL
// body:       optional request body (e.g., struct for POST JSON), pass nil for GET
// headers:    optional headers (e.g., Content-Type, API keys)
// result:     optional pointer to decode JSON response into (pass nil if not needed)
func DoHttpRequestWithLogging(method, label, url string, body interface{}, headers map[string]string, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			logger.Log.Error().Str("label", label).Err(err).Msg("Failed to marshal request body")
			return fmt.Errorf("marshal request failed: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	// Construct request
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		logger.Log.Error().Str("label", label).Err(err).Msg("Failed to create HTTP request")
		return fmt.Errorf("create request failed: %w", err)
	}

	// Set headers if provided
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		logger.Log.Error().
			Str("label", label).
			Str("url", url).
			Str("method", method).
			Dur("duration", duration).
			Err(err).
			Msg("Failed to send HTTP request")
		return fmt.Errorf("send %s failed: %w", label, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error().
			Str("label", label).
			Str("url", url).
			Dur("duration", duration).
			Err(err).
			Msg("Failed to read response body")
		return fmt.Errorf("read response failed for %s: %w", label, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log.Error().
			Str("label", label).
			Str("url", url).
			Str("method", method).
			Int("status_code", resp.StatusCode).
			Dur("duration", duration).
			Msg("Non-200 HTTP status")
		return fmt.Errorf("non-200 response for %s: %d", label, resp.StatusCode)
	}

	logger.Log.Info().
		Str("label", label).
		Str("url", url).
		Str("method", method).
		Int("status_code", resp.StatusCode).
		Int("response_size", len(respBody)).
		Dur("duration", duration).
		Msg("HTTP request completed")

	// Optional: unmarshal into result
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			logger.Log.Error().
				Str("label", label).
				Str("url", url).
				Dur("duration", duration).
				Err(err).
				Msg("Failed to unmarshal response body")
			return fmt.Errorf("unmarshal response failed for %s: %w", label, err)
		}
	}
	return nil
}
