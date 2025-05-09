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

var envFlag = flag.String("env", "local", "environment to run (value must exist in envHosts or 'all')")

var envHosts = map[string]string{
	"local":        "http://127.0.0.1:8080",
	"local-docker": "http://127.0.0.1:8050",
	"dev":          "http://nlb.devops.tantin.com:8000/api/tx-aggregator",
	"test":         "http://aaa:8000/api/tx-aggregator",
	"prod":         "http://tx-aggregator.service.consul:8050",
}

const countsFile = "testcases/expected_counts.txt"

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

func runSuite(baseURL string, paths []string) bool {
	passed := 0
	base, _ := url.Parse(baseURL)
	expectedCounts := loadExpectedCounts()
	updatedCounts := make(map[string]int)

	for idx, p := range paths {
		fullURL := buildFullURL(base, p)
		fmt.Printf("Test #%d: %s\n", idx+1, fullURL)

		firstResp, err := doRequest(fullURL)
		if err != nil {
			fmt.Println("First request error:", err)
			continue
		}

		time.Sleep(500 * time.Millisecond)

		secondResp, err := doRequest(fullURL)
		if err != nil {
			fmt.Println("Second request error:", err)
			continue
		}

		count := extractCount(secondResp)
		relURI := buildFullURL(base, p)[len(base.Scheme+"://"+base.Host):]
		prevCount, exists := expectedCounts[relURI]

		if !exists {
			updatedCounts[relURI] = count
			fmt.Printf("✅ PASS (items: %d) [initial record]\n", count)
			passed++
			continue
		}

		if count < prevCount {
			fmt.Printf("❌ FAIL: item count dropped! current=%d, expected=%d\n", count, prevCount)
			continue
		}

		if !assert.ObjectsAreEqual(firstResp, secondResp) {
			fmt.Println("❌ FAIL: response mismatch")
			printResponseDiff(firstResp, secondResp)
			continue
		}

		updatedCounts[relURI] = count
		fmt.Printf("✅ PASS (items: %d) [prev: %d]\n", count, prevCount)
		passed++
	}

	saveExpectedCounts(updatedCounts)
	fmt.Printf("Summary: %d / %d passed\n", passed, len(paths))
	return passed == len(paths)
}

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

func buildFullURL(base *url.URL, requestURI string) string {
	u, _ := url.Parse(requestURI)
	out := *base
	out.Path = path.Join(base.Path, u.Path)
	out.RawQuery = u.RawQuery
	return out.String()
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

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

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

func saveExpectedCounts(data map[string]int) {
	f, err := os.Create(countsFile)
	if err != nil {
		fmt.Println("Failed to save counts:", err)
		return
	}
	defer f.Close()
	for uri, count := range data {
		fmt.Fprintf(f, "%d %s\n", count, uri)
	}
}

func sortedEnvKeys() []string {
	keys := make([]string, 0, len(envHosts))
	for k := range envHosts {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
