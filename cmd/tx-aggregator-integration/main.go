// File: main.go
//
// A simple integration‑test runner that executes the same set of HTTP test
// cases against one or more deployment environments (local, test, prod).
// The test‑case file contains only the request path + query string, so the
// correct domain is injected at runtime based on the selected environment.
//
// Usage examples:
//
//	go run main.go                    # default: env=local
//	go run main.go -env=test          # run only test environment
//	go run main.go -env=all           # run local + test + prod
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
	"os"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---- command‑line flags -----------------------------------------------------

// envFlag chooses which environment(s) to test: local | test | prod | all
var envFlag = flag.String("env", "local", "environment to run (local|test|prod|all)")

// ---- environment configuration ---------------------------------------------

// envHosts maps environment names to their base URLs.
// Only the hostname / port differs between environments; all paths stay the same.
var envHosts = map[string]string{
	"local":   "http://127.0.0.1:8080",
	"test":    "http://nlb.devops.tantin.com:8000/api/tx-aggregator",
	"prod":    "https://wallet-api.tantin.com/api/tx-aggregator",
	"prod-in": "http://tx-aggregator.service.consul:8050",
}

// ---- entry point ------------------------------------------------------------

func main() {
	flag.Parse()
	fmt.Println("Starting integration tests …")

	paths, err := loadTestCases("testcases/integration_testcases.txt")
	if err != nil {
		fmt.Println("Failed to load test cases:", err)
		os.Exit(1)
	}

	// Resolve which environments to run
	var envs []string
	switch *envFlag {
	case "all":
		envs = []string{"local", "test", "prod"}
	case "local", "test", "prod":
		envs = []string{*envFlag}
	default:
		fmt.Printf("Unknown env %q (valid: local|test|prod|all)\n", *envFlag)
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

// runSuite executes all test cases against a single environment.
// Returns true if all tests passed.
func runSuite(baseURL string, paths []string) bool {
	passed := 0

	for idx, p := range paths {
		fullURL := baseURL + p
		fmt.Printf("Test #%d: %s\n", idx+1, fullURL)

		firstResp, err := doRequest(fullURL)
		if err != nil {
			fmt.Println("First request error:", err)
			continue
		}

		// small delay to give cache / rate‑limit logic a chance to differ
		time.Sleep(500 * time.Millisecond)

		secondResp, err := doRequest(fullURL)
		if err != nil {
			fmt.Println("Second request error:", err)
			continue
		}

		if assert.ObjectsAreEqual(firstResp, secondResp) {
			fmt.Println("✅ PASS")
			passed++
		} else {
			fmt.Println("❌ FAIL: response mismatch")
			printResponseDiff(firstResp, secondResp)
		}
	}

	fmt.Printf("Summary: %d / %d passed\n", passed, len(paths))
	return passed == len(paths)
}

// ---- helpers ----------------------------------------------------------------

// loadTestCases reads the .txt file and returns a slice of request paths.
// Lines beginning with “#” or blank lines are ignored.
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

		// Accept either full URLs or relative paths; convert full URLs to paths.
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			u, err := http.NewRequest(http.MethodGet, line, nil)
			if err != nil {
				return nil, fmt.Errorf("invalid URL in test file: %s", line)
			}
			line = u.URL.RequestURI()
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

// printResponseDiff outputs the key‑wise differences between two JSON objects.
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
