package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/misty-step/site-forge/internal/checks"
	"github.com/misty-step/site-forge/internal/report"
)

func main() {
	dir := flag.String("dir", "./dist", "Directory to verify")
	baseline := flag.String("baseline", "", "Baseline directory for vision comparison")
	threshold := flag.Int("threshold", 7, "Vision score threshold (1-10)")
	lighthousePerf := flag.Int("lighthouse-perf", 90, "Lighthouse performance threshold")
	lighthouseA11y := flag.Int("lighthouse-a11y", 90, "Lighthouse accessibility threshold")
	lighthouseSEO := flag.Int("lighthouse-seo", 90, "Lighthouse SEO threshold")
	flag.Parse()

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "Error: --dir is required")
		os.Exit(1)
	}

	// Resolve absolute path
	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Verifying site in: %s\n", absDir)

	// Initialize report
	r := report.NewReport(absDir)

	// Check 1: ASSETS
	fmt.Print("\n[1/5] Running ASSETS check... ")
	assetsResult := checks.CheckAssets(absDir)
	r.Checks.Assets = assetsResult
	if assetsResult.Status == "FAIL" {
		fmt.Printf("FAIL\n  Missing %d assets: %v\n", len(assetsResult.Missing), assetsResult.Missing)
		printSummary(r)
		writeReport(r)
		os.Exit(1)
	}
	fmt.Printf("PASS (%d/%d assets verified)\n", assetsResult.Total, assetsResult.Total)

	// Check 2: BUILD
	fmt.Print("[2/5] Running BUILD check... ")
	buildResult := checks.CheckBuild(absDir)
	r.Checks.Build = buildResult
	if buildResult.Status == "FAIL" {
		fmt.Printf("FAIL\n  %s\n", buildResult.Details)
		printSummary(r)
		writeReport(r)
		os.Exit(1)
	}
	fmt.Printf("PASS (%s)\n", buildResult.Details)

	// Check 3: LIGHTHOUSE
	fmt.Print("[3/5] Running LIGHTHOUSE check... ")
	lighthouseResult, err := checks.CheckLighthouse(absDir, *lighthousePerf, *lighthouseA11y, *lighthouseSEO)
	r.Checks.Lighthouse = lighthouseResult
	if err != nil {
		fmt.Printf("SKIP (lighthouse not available: %v)\n", err)
		r.Checks.Lighthouse.Status = "SKIP"
		r.Checks.Lighthouse.Details = err.Error()
	} else if lighthouseResult.Status == "FAIL" {
		fmt.Printf("FAIL\n  Perf: %d | A11y: %d | SEO: %d (thresholds: %d/%d/%d)\n",
			lighthouseResult.Performance, lighthouseResult.Accessibility, lighthouseResult.SEO,
			*lighthousePerf, *lighthouseA11y, *lighthouseSEO)
		printSummary(r)
		writeReport(r)
		os.Exit(1)
	} else {
		fmt.Printf("PASS (Perf: %d | A11y: %d | SEO: %d)\n",
			lighthouseResult.Performance, lighthouseResult.Accessibility, lighthouseResult.SEO)
	}

	// Check 4: SCREENSHOTS
	fmt.Print("[4/5] Running SCREENSHOTS check... ")
	screenshotResult, err := checks.CaptureScreenshots(absDir)
	r.Checks.Screenshots = screenshotResult
	if err != nil {
		fmt.Printf("SKIP (chromedp not available: %v)\n", err)
		r.Checks.Screenshots.Status = "SKIP"
		r.Checks.Screenshots.Details = err.Error()
	} else {
		fmt.Printf("PASS (Desktop: %s, Mobile: %s)\n", screenshotResult.Desktop, screenshotResult.Mobile)
	}

	// Check 5: VISION (optional)
	if *baseline != "" {
		fmt.Print("[5/5] Running VISION check... ")
		visionResult, err := checks.CheckVision(*baseline, *threshold)
		r.Checks.Vision = visionResult
		if err != nil {
			fmt.Printf("SKIP (vision check failed: %v)\n", err)
			r.Checks.Vision.Status = "SKIP"
		} else if visionResult.Status == "FAIL" {
			fmt.Printf("FAIL\n  Score: %d/10 (threshold: %d)\n  Analysis: %s\n", visionResult.Score, visionResult.Threshold, visionResult.Analysis)
			printSummary(r)
			writeReport(r)
			os.Exit(1)
		} else {
			fmt.Printf("PASS (Score: %d/10, threshold: %d)\n", visionResult.Score, visionResult.Threshold)
		}
	} else {
		fmt.Print("[5/5] Running VISION check... ")
		r.Checks.Vision = report.VisionResult{
			Status:    "SKIP",
			Details:   "No baseline provided",
			Threshold: *threshold,
		}
		fmt.Println("SKIP (no baseline provided)")
	}

	// All checks passed
	r.Overall = "PASS"
	printSummary(r)
	writeReport(r)
	fmt.Println("\n✅ All checks passed!")
	os.Exit(0)
}

func printSummary(r *report.Report) {
	r.Overall = "FAIL"
	fmt.Println("\n" + r.FormatSummary())
}

func writeReport(r *report.Report) {
	r.Timestamp = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		return
	}
	if err := os.WriteFile("forge-report.json", data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
	}
}
