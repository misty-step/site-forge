package checks

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/misty-step/site-forge/internal/report"
)

// CheckVision compares screenshots with baseline using OpenRouter API
func CheckVision(baselineDir string, threshold int) (report.VisionResult, error) {
	result := report.VisionResult{
		Status:    "PASS",
		Threshold: threshold,
	}

	// Read API key
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return result, fmt.Errorf("OPENROUTER_API_KEY not set")
	}

	// Check for baseline screenshots
	baselineDesktop := filepath.Join(baselineDir, "desktop.png")
	baselineMobile := filepath.Join(baselineDir, "mobile.png")

	if _, err := os.Stat(baselineDesktop); os.IsNotExist(err) {
		return result, fmt.Errorf("baseline desktop.png not found in %s", baselineDir)
	}
	if _, err := os.Stat(baselineMobile); os.IsNotExist(err) {
		return result, fmt.Errorf("baseline mobile.png not found in %s", baselineDir)
	}

	// Check for new screenshots
	newDesktop := "screenshots/desktop.png"
	newMobile := "screenshots/mobile.png"

	if _, err := os.Stat(newDesktop); os.IsNotExist(err) {
		return result, fmt.Errorf("new desktop.png not found (run screenshots check first)")
	}
	if _, err := os.Stat(newMobile); os.IsNotExist(err) {
		return result, fmt.Errorf("new mobile.png not found (run screenshots check first)")
	}

	// Read and encode images
	desktopBase64, err := encodeImage(newDesktop)
	if err != nil {
		return result, fmt.Errorf("failed to encode desktop screenshot: %v", err)
	}

	mobileBase64, err := encodeImage(newMobile)
	if err != nil {
		return result, fmt.Errorf("failed to encode mobile screenshot: %v", err)
	}

	baselineDesktopBase64, err := encodeImage(baselineDesktop)
	if err != nil {
		return result, fmt.Errorf("failed to encode baseline desktop: %v", err)
	}

	baselineMobileBase64, err := encodeImage(baselineMobile)
	if err != nil {
		return result, fmt.Errorf("failed to encode baseline mobile: %v", err)
	}

	// Call OpenRouter API with vision model
	analysis, err := callVisionAPI(apiKey, baselineDesktopBase64, baselineMobileBase64, desktopBase64, mobileBase64)
	if err != nil {
		return result, fmt.Errorf("vision API call failed: %v", err)
	}

	// Parse score from response
	score := parseScoreFromAnalysis(analysis)
	result.Score = score
	result.Analysis = analysis

	if score < threshold {
		result.Status = "FAIL"
	}

	return result, nil
}

func encodeImage(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

type OpenRouterRequest struct {
	Model    string        `json:"model"`
	Messages []Message     `json:"messages"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func callVisionAPI(apiKey, baselineDesktop, baselineMobile, newDesktop, newMobile string) (string, error) {
	prompt := `Compare the original website screenshots (BASELINE) with the redesigned website screenshots (NEW). 

Analyze and score the redesign on a scale of 1-10 for each category:
1. Visual polish (is the redesign more professional, modern, and visually appealing?)
2. Brand fidelity (does it still feel like the same business? Same colors, style, vibe?)
3. Content completeness (is anything from the original missing? Are all sections present?)
4. Mobile experience (is mobile layout better, worse, or about the same?)

Then provide an overall score 1-10 with the question: "Would the business owner be impressed?"

Respond in this exact format:
VISUAL_POLISH: X/10
BRAND_FIDELITY: X/10  
CONTENT_COMPLETENESS: X/10
MOBILE_EXPERIENCE: X/10
OVERALL: X/10
ANALYSIS: [2-3 sentences of specific feedback on what's better and what could improve]`

	req := OpenRouterRequest{
		Model: "anthropic/claude-sonnet-4-20250514",
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: "BASELINE - Desktop:\n" + prompt,
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: "data:image/png;base64," + baselineDesktop,
						},
					},
					{
						Type: "text",
						Text: "BASELINE - Mobile:\n",
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: "data:image/png;base64," + baselineMobile,
						},
					},
					{
						Type: "text",
						Text: "NEW - Desktop:\n",
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: "data:image/png;base64," + newDesktop,
						},
					},
					{
						Type: "text",
						Text: "NEW - Mobile:\n",
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: "data:image/png;base64," + newMobile,
						},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/misty-step/site-forge")
	httpReq.Header.Set("X-Title", "Site Forge")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var orResp OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		return "", err
	}

	if len(orResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return orResp.Choices[0].Message.Content[0].Text, nil
}

func parseScoreFromAnalysis(analysis string) int {
	// Try to find OVERALL: X/10 pattern
	re := regexp.MustCompile(`OVERALL:\s*(\d+)\s*/\s*10`)
	matches := re.FindStringSubmatch(analysis)
	if len(matches) > 1 {
		if score, err := strconv.Atoi(matches[1]); err == nil {
			return score
		}
	}

	// Fallback: look for any number 1-10 in the text
	re = regexp.MustCompile(`\b([1-9]|10)\b`)
	matches = re.FindStringSubmatch(analysis)
	if len(matches) > 1 {
		if score, err := strconv.Atoi(matches[1]); err == nil {
			return score
		}
	}

	// Default to threshold if we can't parse
	return 0
}
