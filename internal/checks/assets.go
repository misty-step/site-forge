package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"github.com/misty-step/site-forge/internal/report"
)

// CheckAssets verifies all referenced assets in HTML files exist
func CheckAssets(distDir string) report.AssetsResult {
	result := report.AssetsResult{
		Status: "PASS",
	}

	// Find all HTML files
	htmlFiles, err := findHTMLFiles(distDir)
	if err != nil {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Error finding HTML files: %v", err)
		return result
	}

	if len(htmlFiles) == 0 {
		result.Status = "FAIL"
		result.Details = "No HTML files found in dist directory"
		return result
	}

	// Extract and verify all assets
	var missing []string
	totalAssets := 0

	for _, htmlFile := range htmlFiles {
		assets, err := extractAssets(htmlFile, distDir)
		if err != nil {
			result.Status = "FAIL"
			result.Details = fmt.Sprintf("Error parsing %s: %v", htmlFile, err)
			return result
		}

		for _, asset := range assets {
			totalAssets++
			// Check if file exists
			// Key insight: on Unix, paths like /images/foo.jpg are URL paths, not absolute filesystem paths
			// filepath.IsAbs returns true for these, but they're actually relative to the web root
			assetPath := asset
			
			// Check if it's a URL-style path (starts with /)
			if strings.HasPrefix(asset, "/") {
				// Path like /images/foo.jpg or /the-farm-house-demo/images/foo.jpg
				// Strip any basePath prefix to find actual file in dist
				relPath := asset
				
				// Find the position of known content paths
				if idx := strings.Index(asset, "/images/"); idx >= 0 {
					relPath = asset[idx:]
				} else if idx := strings.Index(asset, "/_astro/"); idx >= 0 {
					relPath = asset[idx:]
				} else if strings.HasPrefix(asset, "/favicon") {
					// Keep favicon as-is
					relPath = asset
				} else {
					// Default: use as-is
					relPath = asset
				}
				
				// Concatenate to avoid filepath.Join ignoring the base on Unix
				assetPath = distDir + relPath
			} else if strings.HasPrefix(asset, "./") {
				// Relative path starting with ./
				assetPath = filepath.Join(distDir, asset[1:]) // Remove the leading .
			} else {
				// Regular relative path
				assetPath = filepath.Join(distDir, asset)
			}
			
			if _, err := os.Stat(assetPath); os.IsNotExist(err) {
				missing = append(missing, asset)
			}
		}
	}

	result.Total = totalAssets
	result.Missing = missing

	if len(missing) > 0 {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Missing %d assets", len(missing))
	} else {
		result.Details = fmt.Sprintf("%d/%d assets verified", totalAssets, totalAssets)
	}

	return result
}

// findHTMLFiles recursively finds all HTML files in a directory
func findHTMLFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			subFiles, err := findHTMLFiles(path)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else if strings.HasSuffix(entry.Name(), ".html") || strings.HasSuffix(entry.Name(), ".htm") {
			files = append(files, path)
		}
	}

	return files, nil
}

// extractAssets extracts img src, link href, and script src from an HTML file
func extractAssets(htmlFile, baseDir string) ([]string, error) {
	data, err := os.ReadFile(htmlFile)
	if err != nil {
		return nil, err
	}

	assets := make([]string, 0)
	
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		// Check img tags for src
		if n.Type == html.ElementNode {
			switch n.Data {
			case "img":
				for _, attr := range n.Attr {
					if attr.Key == "src" && attr.Val != "" && !strings.HasPrefix(attr.Val, "data:") {
						assets = append(assets, attr.Val)
					}
				}
			case "link":
				for _, attr := range n.Attr {
					if attr.Key == "href" && attr.Val != "" {
						// Only include stylesheets and icons
						for _, a := range n.Attr {
							if a.Key == "rel" && (a.Val == "stylesheet" || a.Val == "icon" || a.Val == "shortcut") {
								assets = append(assets, attr.Val)
								break
							}
						}
					}
				}
			case "script":
				for _, attr := range n.Attr {
					if attr.Key == "src" && attr.Val != "" {
						assets = append(assets, attr.Val)
					}
				}
			case "source":
				// Handle <source> tags in <picture> elements
				for _, attr := range n.Attr {
					if attr.Key == "srcset" && attr.Val != "" {
						// Parse srcset - could have multiple sources
						parts := strings.Split(attr.Val, ",")
						for _, part := range parts {
							src := strings.TrimSpace(strings.Split(part, " ")[0])
							if src != "" {
								assets = append(assets, src)
							}
						}
					}
				}
			}
		}
		
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)
	return assets, nil
}
