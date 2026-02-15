package checks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindHTMLFiles(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "site-forge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html></html>"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "about.html"), []byte("<html></html>"), 0644)

	files, err := findHTMLFiles(tmpDir)
	if err != nil {
		t.Fatalf("findHTMLFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 HTML files, got %d", len(files))
	}
}

func TestExtractAssets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-forge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	htmlContent := `<!DOCTYPE html>
<html>
<head>
	<link rel="stylesheet" href="style.css">
	<link rel="icon" href="favicon.ico">
</head>
<body>
	<img src="hero.jpg" alt="Hero">
	<img src="images/logo.svg" alt="Logo">
	<script src="app.js"></script>
	<picture>
		<source srcset="banner.webp" type="image/webp">
		<source srcset="banner.jpg" type="image/jpeg">
		<img src="banner-fallback.jpg" alt="Banner">
	</picture>
</body>
</html>`

	htmlFile := filepath.Join(tmpDir, "index.html")
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create the referenced files
	os.MkdirAll(filepath.Join(tmpDir, "images"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte("body {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "favicon.ico"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "hero.jpg"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "images", "logo.svg"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "app.js"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "banner.webp"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "banner.jpg"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "banner-fallback.jpg"), []byte{}, 0644)

	assets, err := extractAssets(htmlFile, tmpDir)
	if err != nil {
		t.Fatalf("extractAssets failed: %v", err)
	}

	expected := []string{
		"style.css",
		"favicon.ico",
		"hero.jpg",
		"images/logo.svg",
		"app.js",
		"banner.webp",
		"banner.jpg",
		"banner-fallback.jpg",
	}

	if len(assets) != len(expected) {
		t.Errorf("Expected %d assets, got %d", len(expected), len(assets))
		t.Logf("Got: %v", assets)
	}
}

func TestCheckAssets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-forge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create HTML with valid assets
	htmlContent := `<!DOCTYPE html>
<html>
<head><link rel="stylesheet" href="style.css"></head>
<body><img src="hero.jpg"><script src="app.js"></script></body>
</html>`

	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(htmlContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte("body {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "app.js"), []byte("console.log()"), 0644)
	// Note: hero.jpg is missing

	result := CheckAssets(tmpDir)

	if result.Status != "FAIL" {
		t.Errorf("Expected FAIL status for missing asset, got %s", result.Status)
	}

	if len(result.Missing) != 1 || result.Missing[0] != "hero.jpg" {
		t.Errorf("Expected missing hero.jpg, got %v", result.Missing)
	}
}

func TestCheckAssetsAllValid(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-forge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	htmlContent := `<!DOCTYPE html>
<html>
<head><link rel="stylesheet" href="style.css"></head>
<body><img src="hero.jpg"><script src="app.js"></script></body>
</html>`

	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(htmlContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte("body {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "app.js"), []byte("console.log()"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "hero.jpg"), []byte{}, 0644)

	result := CheckAssets(tmpDir)

	if result.Status != "PASS" {
		t.Errorf("Expected PASS status, got %s: %s", result.Status, result.Details)
	}

	if result.Total != 3 { // style.css, hero.jpg, app.js
		t.Errorf("Expected 3 assets, got %d", result.Total)
	}
}

func TestCheckBuild(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-forge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test missing index.html
	result := CheckBuild(tmpDir)
	if result.Status != "FAIL" {
		t.Errorf("Expected FAIL for missing index.html, got %s", result.Status)
	}

	// Test valid HTML
	validHTML := `<!DOCTYPE html>
<html lang="en">
<head>
	<title>Test Site</title>
	<meta name="description" content="A test site">
	<meta property="og:title" content="Test Site">
</head>
<body><h1>Hello</h1></body>
</html>`

	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(validHTML), 0644)

	result = CheckBuild(tmpDir)
	if result.Status != "PASS" {
		t.Errorf("Expected PASS for valid HTML, got %s: %s", result.Status, result.Details)
	}

	if result.Pages != 1 {
		t.Errorf("Expected 1 page, got %d", result.Pages)
	}
}

func TestCheckBuildMissingMeta(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-forge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// HTML missing meta tags
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello</h1></body>
</html>`

	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(htmlContent), 0644)

	result := CheckBuild(tmpDir)
	if result.Status != "FAIL" {
		t.Errorf("Expected FAIL for missing meta tags, got %s", result.Status)
	}
}
