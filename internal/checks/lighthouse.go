package checks

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/misty-step/site-forge/internal/report"
)

// CheckLighthouse runs Lighthouse audit on the dist directory
func CheckLighthouse(distDir string, perfThreshold, a11yThreshold, seoThreshold int) (report.LighthouseResult, error) {
	result := report.LighthouseResult{
		Status: "PASS",
		Thresholds: report.Thresholds{
			Performance:   perfThreshold,
			Accessibility: a11yThreshold,
			SEO:           seoThreshold,
		},
	}

	// Check if lighthouse is available
	if !isLighthouseAvailable() {
		return result, fmt.Errorf("lighthouse not installed (run: npm install -g lighthouse)")
	}

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		return result, fmt.Errorf("failed to find available port: %v", err)
	}

	// Start a local server
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: http.FileServer(http.Dir(distDir)),
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// Run lighthouse
	url := fmt.Sprintf("http://localhost:%d", port)
	lighthouseScores, err := runLighthouse(url)
	
	// Shutdown server
	server.Close()

	if err != nil {
		return result, fmt.Errorf("lighthouse failed: %v", err)
	}

	result.Performance = lighthouseScores.Performance
	result.Accessibility = lighthouseScores.Accessibility
	result.SEO = lighthouseScores.SEO

	// Check thresholds
	if result.Performance < perfThreshold || result.Accessibility < a11yThreshold || result.SEO < seoThreshold {
		result.Status = "FAIL"
	}

	result.Details = fmt.Sprintf("Perf: %d, A11y: %d, SEO: %d", result.Performance, result.Accessibility, result.SEO)

	return result, nil
}

func isLighthouseAvailable() bool {
	cmd := exec.Command("npx", "lighthouse", "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "lighthouse")
}

func findAvailablePort() (int, error) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()

	// Extract port from the address
	addr := ln.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

type lighthouseScores struct {
	Performance  int
	Accessibility int
	SEO          int
}

type lighthouseJSON struct {
	Categories struct {
		Performance struct {
			Score float64 `json:"score"`
		} `json:"performance"`
		Accessibility struct {
			Score float64 `json:"score"`
		} `json:"accessibility"`
		SEO struct {
			Score float64 `json:"score"`
		} `json:"seo"`
	} `json:"categories"`
}

func runLighthouse(url string) (lighthouseScores, error) {
	// Create temp file for JSON output
	tmpFile, err := os.CreateTemp("", "lighthouse-*.json")
	if err != nil {
		return lighthouseScores{}, err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Run lighthouse
	cmd := exec.Command(
		"npx", "lighthouse", url,
		"--output=json",
		"--output-path="+tmpPath,
		"--chrome-flags=--headless --no-sandbox --disable-gpu",
		"--quiet",
		"--only-categories=performance,accessibility,seo",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a timeout or other issue
		if strings.Contains(string(output), "timeout") {
			return lighthouseScores{}, fmt.Errorf("lighthouse timeout")
		}
		return lighthouseScores{}, fmt.Errorf("lighthouse error: %v, output: %s", err, string(output))
	}

	// Read and parse result
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return lighthouseScores{}, fmt.Errorf("failed to read lighthouse output: %v", err)
	}

	var result lighthouseJSON
	if err := json.Unmarshal(data, &result); err != nil {
		return lighthouseScores{}, fmt.Errorf("failed to parse lighthouse JSON: %v", err)
	}

	// Convert scores from 0-1 to 0-100
	perfScore := int(result.Categories.Performance.Score * 100)
	a11yScore := int(result.Categories.Accessibility.Score * 100)
	seoScore := int(result.Categories.SEO.Score * 100)

	return lighthouseScores{
		Performance:   perfScore,
		Accessibility: a11yScore,
		SEO:          seoScore,
	}, nil
}

// Ensure the report package has the correct structure for lighthouse
func init() {
	_ = filepath.Join("", "")
}
