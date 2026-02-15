package report

import (
	"fmt"
)

type Report struct {
	Timestamp string      `json:"timestamp"`
	Directory string      `json:"directory"`
	Overall   string      `json:"overall"`
	Checks    ReportChecks `json:"checks"`
}

type ReportChecks struct {
	Assets     AssetsResult     `json:"assets"`
	Build      BuildResult       `json:"build"`
	Lighthouse LighthouseResult  `json:"lighthouse"`
	Screenshots ScreenshotsResult `json:"screenshots"`
	Vision     VisionResult      `json:"vision"`
}

func NewReport(dir string) *Report {
	return &Report{
		Directory: dir,
		Overall:   "FAIL",
		Checks: ReportChecks{
			Assets:     AssetsResult{Status: "FAIL"},
			Build:      BuildResult{Status: "FAIL"},
			Lighthouse: LighthouseResult{Status: "FAIL"},
			Screenshots: ScreenshotsResult{Status: "FAIL"},
			Vision:     VisionResult{Status: "SKIP"},
		},
	}
}

func (r *Report) FormatSummary() string {
	summary := "site-forge verify results:\n"

	// Assets
	if r.Checks.Assets.Status == "PASS" {
		summary += fmt.Sprintf("  ✅ ASSETS: %d/%d assets verified\n", r.Checks.Assets.Total, r.Checks.Assets.Total)
	} else {
		summary += fmt.Sprintf("  ❌ ASSETS: FAIL - Missing %d assets\n", len(r.Checks.Assets.Missing))
	}

	// Build
	if r.Checks.Build.Status == "PASS" {
		summary += fmt.Sprintf("  ✅ BUILD: %s\n", r.Checks.Build.Details)
	} else {
		summary += fmt.Sprintf("  ❌ BUILD: FAIL - %s\n", r.Checks.Build.Details)
	}

	// Lighthouse
	if r.Checks.Lighthouse.Status == "PASS" {
		summary += fmt.Sprintf("  ✅ LIGHTHOUSE: Perf %d | A11y %d | SEO %d\n",
			r.Checks.Lighthouse.Performance, r.Checks.Lighthouse.Accessibility, r.Checks.Lighthouse.SEO)
	} else if r.Checks.Lighthouse.Status == "SKIP" {
		summary += fmt.Sprintf("  ⚠️  LIGHTHOUSE: SKIP - %s\n", r.Checks.Lighthouse.Details)
	} else {
		summary += fmt.Sprintf("  ❌ LIGHTHOUSE: Perf %d | A11y %d | SEO %d (thresholds not met)\n",
			r.Checks.Lighthouse.Performance, r.Checks.Lighthouse.Accessibility, r.Checks.Lighthouse.SEO)
	}

	// Screenshots
	if r.Checks.Screenshots.Status == "PASS" {
		summary += fmt.Sprintf("  ✅ SCREENSHOTS: Desktop + Mobile captured\n")
	} else if r.Checks.Screenshots.Status == "SKIP" {
		summary += fmt.Sprintf("  ⚠️  SCREENSHOTS: SKIP - %s\n", r.Checks.Screenshots.Details)
	} else {
		summary += fmt.Sprintf("  ❌ SCREENSHOTS: FAIL - %s\n", r.Checks.Screenshots.Details)
	}

	// Vision
	if r.Checks.Vision.Status == "PASS" {
		summary += fmt.Sprintf("  ✅ VISION: Score %d/10 (threshold: %d)\n", r.Checks.Vision.Score, r.Checks.Vision.Threshold)
	} else if r.Checks.Vision.Status == "SKIP" {
		summary += fmt.Sprintf("  ⚠️  VISION: SKIP - %s\n", r.Checks.Vision.Details)
	} else if r.Checks.Vision.Status == "FAIL" {
		summary += fmt.Sprintf("  ❌ VISION: Score %d/10 (threshold: %d) - %s\n", r.Checks.Vision.Score, r.Checks.Vision.Threshold, r.Checks.Vision.Analysis)
	}

	summary += fmt.Sprintf("\nOVERALL: %s\n", r.Overall)
	if r.Overall == "PASS" {
		summary += "✅"
	} else {
		summary += "❌"
	}

	return summary
}

type AssetsResult struct {
	Status  string   `json:"status"`
	Total   int      `json:"total"`
	Missing []string `json:"missing,omitempty"`
	Details string   `json:"details"`
}

type BuildResult struct {
	Status  string `json:"status"`
	Pages   int    `json:"pages"`
	Details string `json:"details"`
}

type LighthouseResult struct {
	Status       string     `json:"status"`
	Performance  int        `json:"performance"`
	Accessibility int       `json:"accessibility"`
	SEO          int        `json:"seo"`
	Thresholds   Thresholds `json:"thresholds"`
	Details      string     `json:"details,omitempty"`
}

type Thresholds struct {
	Performance   int `json:"performance"`
	Accessibility int `json:"accessibility"`
	SEO           int `json:"seo"`
}

type ScreenshotsResult struct {
	Status  string `json:"status"`
	Desktop string `json:"desktop,omitempty"`
	Mobile  string `json:"mobile,omitempty"`
	Details string `json:"details,omitempty"`
}

type VisionResult struct {
	Status    string `json:"status"`
	Score     int    `json:"score,omitempty"`
	Threshold int    `json:"threshold"`
	Analysis  string `json:"analysis,omitempty"`
	Details   string `json:"details,omitempty"`
}
