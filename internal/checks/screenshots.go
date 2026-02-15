package checks

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/misty-step/site-forge/internal/report"
)

// CaptureScreenshots captures desktop and mobile screenshots using chromedp
func CaptureScreenshots(distDir string) (report.ScreenshotsResult, error) {
	result := report.ScreenshotsResult{
		Status: "PASS",
	}

	// Create screenshots directory
	screenshotsDir := "screenshots"
	if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Failed to create screenshots directory: %v", err)
		return result, err
	}

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Failed to find available port: %v", err)
		return result, err
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

	url := fmt.Sprintf("http://localhost:%d", port)

	// Create context with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Desktop: allocate chrome with headless mode
	desktopOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	)

	desktopAllocCtx, desktopCancelAlloc := chromedp.NewExecAllocator(ctx, desktopOpts...)
	defer desktopCancelAlloc()

	desktopTaskCtx, desktopCancelTask := chromedp.NewContext(desktopAllocCtx)
	defer desktopCancelTask()

	// Desktop screenshot (1280x900)
	desktopPath := filepath.Join(screenshotsDir, "desktop.png")
	var desktopBuf []byte
	if err := chromedp.Run(desktopTaskCtx,
		chromedp.EmulateViewport(1280, 900),
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // Wait for any animations
		chromedp.FullScreenshot(&desktopBuf, 100),
	); err != nil {
		server.Close()
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Desktop screenshot failed: %v", err)
		return result, err
	}

	if err := os.WriteFile(desktopPath, desktopBuf, 0644); err != nil {
		server.Close()
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Failed to write desktop screenshot: %v", err)
		return result, err
	}

	// Mobile: allocate chrome with mobile user agent
	mobileOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.UserAgent("Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1"),
	)

	mobileAllocCtx, mobileCancelAlloc := chromedp.NewExecAllocator(ctx, mobileOpts...)
	defer mobileCancelAlloc()

	mobileTaskCtx, mobileCancelTask := chromedp.NewContext(mobileAllocCtx)
	defer mobileCancelTask()

	// Mobile screenshot (390x844)
	mobilePath := filepath.Join(screenshotsDir, "mobile.png")
	var mobileBuf []byte
	if err := chromedp.Run(mobileTaskCtx,
		chromedp.EmulateViewport(390, 844),
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.FullScreenshot(&mobileBuf, 100),
	); err != nil {
		server.Close()
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Mobile screenshot failed: %v", err)
		return result, err
	}

	if err := os.WriteFile(mobilePath, mobileBuf, 0644); err != nil {
		server.Close()
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Failed to write mobile screenshot: %v", err)
		return result, err
	}

	// Shutdown server
	server.Close()

	result.Desktop = desktopPath
	result.Mobile = mobilePath

	return result, nil
}
