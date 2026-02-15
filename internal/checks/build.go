package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"github.com/misty-step/site-forge/internal/report"
)

// CheckBuild verifies the HTML build is valid
func CheckBuild(distDir string) report.BuildResult {
	result := report.BuildResult{
		Status: "PASS",
	}

	// Check for index.html
	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		result.Status = "FAIL"
		result.Details = "index.html not found"
		return result
	}

	// Read and parse HTML
	data, err := os.ReadFile(indexPath)
	if err != nil {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Failed to read index.html: %v", err)
		return result
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("HTML parse error: %v", err)
		return result
	}

	// Check for basic structure
	hasHTML := false
	hasHead := false
	hasBody := false
	hasTitle := false
	hasDescription := false
	hasOgTitle := false

	var check func(*html.Node)
	check = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "html":
				hasHTML = true
			case "head":
				hasHead = true
			case "body":
				hasBody = true
			case "title":
				hasTitle = true
			case "meta":
				var name, prop, content string
				for _, attr := range n.Attr {
					if attr.Key == "name" {
						name = attr.Val
					}
					if attr.Key == "property" {
						prop = attr.Val
					}
					if attr.Key == "content" {
						content = attr.Val
					}
				}
				if name == "description" && content != "" {
					hasDescription = true
				}
				if prop == "og:title" && content != "" {
					hasOgTitle = true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			check(c)
		}
	}

	check(doc)

	// Validate structure
	var errors []string
	if !hasHTML {
		errors = append(errors, "missing <html> tag")
	}
	if !hasHead {
		errors = append(errors, "missing <head> tag")
	}
	if !hasBody {
		errors = append(errors, "missing <body> tag")
	}
	if !hasTitle {
		errors = append(errors, "missing <title> tag")
	}
	if !hasDescription {
		errors = append(errors, "missing meta description")
	}
	if !hasOgTitle {
		errors = append(errors, "missing og:title meta tag")
	}

	// Count total pages
	htmlFiles, _ := findHTMLFiles(distDir)
	result.Pages = len(htmlFiles)

	if len(errors) > 0 {
		result.Status = "FAIL"
		result.Details = strings.Join(errors, ", ")
		return result
	}

	result.Details = fmt.Sprintf("Valid HTML, %d page(s), meta tags present", result.Pages)
	return result
}
