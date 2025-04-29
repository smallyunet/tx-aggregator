package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
)

func main() {
	fmt.Println("Starting Integration Tests (txt based)...")

	testCases, err := loadTestCases("testcases/integration_testcases.txt")
	if err != nil {
		fmt.Println("Failed to load test cases:", err)
		os.Exit(1)
	}

	passed := 0

	for idx, url := range testCases {
		fmt.Printf("Running test #%d: %s\n", idx+1, url)

		firstResp, err := doRequest(url)
		if err != nil {
			fmt.Println("First request error:", err)
			continue
		}

		time.Sleep(500 * time.Millisecond)

		secondResp, err := doRequest(url)
		if err != nil {
			fmt.Println("Second request error:", err)
			continue
		}

		if assert.ObjectsAreEqual(firstResp, secondResp) {
			fmt.Println("✅ PASS")
			passed++
		} else {
			fmt.Println("❌ FAIL: Response mismatch")
			printResponseDiff(firstResp, secondResp)
		}
	}

	fmt.Printf("\nIntegration Test Summary: Passed %d/%d cases.\n", passed, len(testCases))

	if passed != len(testCases) {
		os.Exit(1)
	}
}

func loadTestCases(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cases []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cases = append(cases, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cases, nil
}

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

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// 🔥 printResponseDiff prints only the differences between two JSON responses.
func printResponseDiff(first, second map[string]interface{}) {
	fmt.Println("--- Differences between first and second response ---")

	for key, firstVal := range first {
		secondVal, exists := second[key]
		if !exists {
			fmt.Printf("Key '%s' missing in second response. First: %v\n", key, firstVal)
			continue
		}
		if !assert.ObjectsAreEqual(firstVal, secondVal) {
			fmt.Printf("Key '%s' differs:\n  First:  %v\n  Second: %v\n", key, firstVal, secondVal)
		}
	}

	for key := range second {
		if _, exists := first[key]; !exists {
			fmt.Printf("Key '%s' missing in first response. Second: %v\n", key, second[key])
		}
	}
}
