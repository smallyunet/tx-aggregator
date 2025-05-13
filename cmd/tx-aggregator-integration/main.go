// Package main provides an integration testing tool for the transaction aggregator service.
// It compares API responses across different environments and tracks expected transaction counts.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
)

// envFlag defines the environment to run tests against (local, local-docker, dev, test, prod, or all)
var envFlag = flag.String("env", "local", "environment to run (value must exist in envHosts or 'all')")

// envHosts maps environment names to their corresponding base URLs
var envHosts = map[string]string{
	"local":        "http://127.0.0.1:8080",
	"local-docker": "http://127.0.0.1:8050",
	"dev":          "http://tx-aggregator.service.consul:8050",
	"test":         "http://tx-aggregator.service.consul:8050",
	"prod":         "http://tx-aggregator.service.consul:8050",
}

// countsFile is the path to the file storing expected transaction counts for each endpoint
const countsFile = "testcases/expected_counts.txt"

// main is the entry point for the integration test tool.
// It loads test cases, resolves environments, and runs test suites for each environment.
func main() {
	flag.Parse()
	fmt.Println("Starting integration tests …")

	// Load test cases from file
	paths, err := loadTestCases("testcases/integration_testcases.txt")
	if err != nil {
		fmt.Println("Failed to load test cases:", err)
		os.Exit(1)
	}

	// Resolve environments based on the provided flag
	envs, err := resolveEnvs(*envFlag)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Run test suites for each environment
	exitCode := 0
	for _, env := range envs {
		fmt.Printf("\n=== Environment: %s (%s) ===\n", env, envHosts[env])
		if !runSuite(envHosts[env], paths) {
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

// resolveEnvs converts the environment flag value to a list of environment names.
// If flagValue is "all", it returns all available environments.
// If flagValue is a valid environment name, it returns that environment.
// Otherwise, it returns an error.
func resolveEnvs(flagValue string) ([]string, error) {
	if flagValue == "all" {
		keys := make([]string, 0, len(envHosts))
		for k := range envHosts {
			keys = append(keys, k)
		}
		sortStrings(keys)
		return keys, nil
	}
	if _, ok := envHosts[flagValue]; ok {
		return []string{flagValue}, nil
	}
	return nil, fmt.Errorf("unknown env %q (valid values: %s or 'all')", flagValue, strings.Join(sortedEnvKeys(), "|"))
}

// runSuite executes the test suite against the specified base URL with the given test paths.
// It makes two requests to each endpoint, compares the responses, and checks transaction counts.
// Returns true if all tests pass, false otherwise.
func runSuite(baseURL string, paths []string) bool {
	passed := 0
	base, _ := url.Parse(baseURL)
	expectedCounts := loadExpectedCounts()
	updatedCounts := make(map[string]int)
	for k, v := range expectedCounts {
		updatedCounts[k] = v
	}

	for idx, p := range paths {
		fullURL := buildFullURL(base, p)
		fmt.Printf("Test #%d: %s\n", idx+1, fullURL)

		// Make first request
		firstResp, err := doRequest(fullURL)
		if err != nil {
			fmt.Println("First request error:", err)
			continue
		}

		// Wait before making second request to ensure consistency
		time.Sleep(500 * time.Millisecond)

		// Make second request
		secondResp, err := doRequest(fullURL)
		if err != nil {
			fmt.Println("Second request error:", err)
			continue
		}

		// Extract transaction count and relative URI
		count := extractCount(secondResp)
		relURI := buildFullURL(base, p)[len(base.Scheme+"://"+base.Host):]
		prevCount, exists := expectedCounts[relURI]

		// Handle case when this is the first time testing this endpoint
		if !exists {
			updatedCounts[relURI] = count
			fmt.Printf("✅ PASS (items: %d) [initial record]\n", count)
			passed++
			continue
		}

		// Fail if transaction count has decreased
		if count < prevCount {
			fmt.Printf("❌ FAIL: item count dropped! current=%d, expected=%d\n", count, prevCount)
			continue
		}

		// Fail if responses don't match
		if !assert.ObjectsAreEqual(firstResp, secondResp) {
			fmt.Println("❌ FAIL: response mismatch")
			printResponseDiff(firstResp, secondResp)
			continue
		}

		// Test passed
		updatedCounts[relURI] = count
		fmt.Printf("✅ PASS (items: %d) [prev: %d]\n", count, prevCount)
		passed++
	}

	// Save updated counts if at least one test passed
	if passed > 0 {
		saveExpectedCounts(updatedCounts)
	} else {
		fmt.Println("No tests passed, will not update expected_counts.txt")
	}

	fmt.Printf("Summary: %d / %d passed\n", passed, len(paths))
	return passed == len(paths)
}

// loadTestCases reads test case URLs from the specified file.
// It skips empty lines and comments (lines starting with #).
// For full URLs, it extracts just the request URI part.
// Returns a list of test case paths/URIs.
func loadTestCases(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var list []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			req, err := http.NewRequest(http.MethodGet, line, nil)
			if err != nil {
				return nil, fmt.Errorf("invalid URL in test file: %s", line)
			}
			line = req.URL.RequestURI()
		}
		list = append(list, line)
	}
	return list, scanner.Err()
}

// buildFullURL combines a base URL with a request URI to create a complete URL.
// It preserves the query parameters from the request URI.
func buildFullURL(base *url.URL, requestURI string) string {
	u, _ := url.Parse(requestURI)
	out := *base
	out.Path = path.Join(base.Path, u.Path)
	out.RawQuery = u.RawQuery
	return out.String()
}

// doRequest performs an HTTP GET request to the specified URL and returns the JSON response.
// It returns an error if the request fails or if the response status is not 200 OK.
func doRequest(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// extractCount extracts the number of transactions from a response payload.
// It navigates through the JSON structure to find the transactions array and returns its length.
// Returns 0 if the transactions array is not found or if any intermediate field is missing.
func extractCount(m map[string]interface{}) int {
	result, ok := m["result"]
	if !ok {
		return 0
	}
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return 0
	}
	txs, ok := resultMap["transactions"]
	if !ok {
		return 0
	}
	if txSlice, ok := txs.([]interface{}); ok {
		return len(txSlice)
	}
	return 0
}

// printResponseDiff prints the differences between two response payloads.
// It identifies keys that are missing in either response and keys with different values.
func printResponseDiff(first, second map[string]interface{}) {
	fmt.Println("--- Differences between first and second response ---")
	// Check for keys in first response
	for key, firstVal := range first {
		secondVal, exists := second[key]
		if !exists {
			fmt.Printf("Key '%s' missing in second response (first = %v)\n", key, firstVal)
			continue
		}
		if !assert.ObjectsAreEqual(firstVal, secondVal) {
			fmt.Printf("Key '%s' differs:\n  First:  %v\n  Second: %v\n", key, firstVal, secondVal)
		}
	}
	// Check for keys in second response that are not in first
	for key := range second {
		if _, exists := first[key]; !exists {
			fmt.Printf("Key '%s' missing in first response (second = %v)\n", key, second[key])
		}
	}
}

// loadExpectedCounts loads the expected transaction counts from the counts file.
// Each line in the file should be in the format: "<count> <uri>".
// Returns a map where keys are URIs and values are the expected transaction counts.
func loadExpectedCounts() map[string]int {
	data := make(map[string]int)
	f, err := os.Open(countsFile)
	if err != nil {
		return data
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		count, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		data[parts[1]] = count
	}
	return data
}

// saveExpectedCounts saves the transaction counts to the counts file.
// Each line in the file will be in the format: "<count> <uri>".
func saveExpectedCounts(data map[string]int) {
	f, err := os.Create(countsFile)
	if err != nil {
		fmt.Println("Failed to save counts:", err)
		return
	}
	defer f.Close()

	// Sort URIs before writing
	uris := make([]string, 0, len(data))
	for uri := range data {
		uris = append(uris, uri)
	}
	sortStrings(uris)

	for _, uri := range uris {
		fmt.Fprintf(f, "%d %s\n", data[uri], uri)
	}
}

// sortedEnvKeys returns a sorted list of environment names from the envHosts map.
// This is used to display valid environment options in error messages.
func sortedEnvKeys() []string {
	keys := make([]string, 0, len(envHosts))
	for k := range envHosts {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

// sortStrings sorts a slice of strings in ascending order using a simple bubble sort algorithm.
// This is used for consistent output in error messages and environment listings.
func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
