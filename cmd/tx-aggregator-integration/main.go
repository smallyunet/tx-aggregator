// File: main.go
//
// A simple integration-test runner that executes the same set of HTTP test
// cases against one or more deployment environments (local, test, prod …).
// The test-case file may contain either relative paths or full URLs that start
// with the local base URL (e.g. http://127.0.0.1:8080/…); the scheme/host are
// discarded automatically.
//
// Usage examples:
//
//	go run main.go                  # default: env=local
//	go run main.go -env=test        # run only “test”
//	go run main.go -env=all         # run all environments in envHosts
//
// Exit status 0  – all environments passed
// Exit status 1+ – at least one environment failed
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
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Command-line flags

var envFlag = flag.String("env", "local",
	"environment to run (value must exist in envHosts or ‘all’)")

// ---------------------------------------------------------------------------
// Environment configuration

// Only scheme / host(/port) / optional prefix differ between environments.
var envHosts = map[string]string{
	"local": "http://127.0.0.1:8080",
	"test":  "http://nlb.devops.tantin.com:8000/api/tx-aggregator",
	"prod":  "http://tx-aggregator.service.consul:8050",
}

// ---------------------------------------------------------------------------
// Entry point

func main() {
	flag.Parse()
	fmt.Println("Starting integration tests …")

	paths, err := loadTestCases("testcases/integration_testcases.txt")
	if err != nil {
		fmt.Println("Failed to load test cases:", err)
		os.Exit(1)
	}

	envs, err := resolveEnvs(*envFlag)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	exitCode := 0
	for _, env := range envs {
		fmt.Printf("\n=== Environment: %s (%s) ===\n", env, envHosts[env])
		if !runSuite(envHosts[env], paths) {
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

// resolveEnvs validates the flag value and returns the list of env names.
func resolveEnvs(flagValue string) ([]string, error) {
	if flagValue == "all" {
		// Collect every key from envHosts deterministically (lexicographic).
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
	return nil, fmt.Errorf("unknown env %q (valid values: %s or ‘all’)",
		flagValue, strings.Join(sortedEnvKeys(), "|"))
}

// ---------------------------------------------------------------------------
// Test-suite execution

func runSuite(baseURL string, paths []string) bool {
	passed := 0
	base, _ := url.Parse(baseURL)

	for idx, p := range paths {
		fullURL := buildFullURL(base, p)
		fmt.Printf("Test #%d: %s\n", idx+1, fullURL)

		firstResp, err := doRequest(fullURL)
		if err != nil {
			fmt.Println("First request error:", err)
			continue
		}

		// Small delay to let cache / rate-limit logic diverge if it would.
		time.Sleep(500 * time.Millisecond)

		secondResp, err := doRequest(fullURL)
		if err != nil {
			fmt.Println("Second request error:", err)
			continue
		}

		if assert.ObjectsAreEqual(firstResp, secondResp) {
			fmt.Printf("✅ PASS (items: %d)\n", extractCount(firstResp))
			passed++
		} else {
			fmt.Println("❌ FAIL: response mismatch")
			printResponseDiff(firstResp, secondResp)
		}
	}

	fmt.Printf("Summary: %d / %d passed\n", passed, len(paths))
	return passed == len(paths)
}

// buildFullURL merges baseURL with requestURI produced by loadTestCases.
func buildFullURL(base *url.URL, requestURI string) string {
	u, _ := url.Parse(requestURI) // requestURI is always absolute-path+query
	out := *base                  // copy
	out.Path = path.Join(base.Path, u.Path)
	out.RawQuery = u.RawQuery
	return out.String()
}

// ---------------------------------------------------------------------------
// Helpers

// loadTestCases reads the .txt file and returns a slice of request URIs
// (leading “/…?query”, no scheme/host), ignoring blank and comment lines.
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

		// Accept either full URLs or relative paths; convert full URLs to URI.
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

// doRequest performs an HTTP GET and decodes the JSON body into a map.
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

// extractCount returns the number of transactions inside result.transactions.
func extractCount(m map[string]interface{}) int {
	result, ok := m["result"]
	if !ok {
		return 0
	}

	// Assert `result` is a map
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return 0
	}

	// Get `transactions` field
	txs, ok := resultMap["transactions"]
	if !ok {
		return 0
	}

	// Check if it's a slice
	if txSlice, ok := txs.([]interface{}); ok {
		return len(txSlice)
	}
	return 0
}

// printResponseDiff outputs the keywise differences between two JSON objects.
func printResponseDiff(first, second map[string]interface{}) {
	fmt.Println("--- Differences between first and second response ---")

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

	for key := range second {
		if _, exists := first[key]; !exists {
			fmt.Printf("Key '%s' missing in first response (second = %v)\n", key, second[key])
		}
	}
}

// ---------------------------------------------------------------------------
// Small utility helpers

func sortedEnvKeys() []string {
	keys := make([]string, 0, len(envHosts))
	for k := range envHosts {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

func sortStrings(s []string) { // small inline sorter to avoid pulling in sort pkg
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
