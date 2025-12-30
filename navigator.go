package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"jaytaylor.com/html2text"
)

// URL regex to extract URLs from text
var urlRegex = regexp.MustCompile(`https?://[^\s<>"'\)]+`)

const NavigatorSystemPrompt = `You are navigating a congressional website to find the page with district office addresses and phone numbers.

Your goal is to find the URL of the page that lists the representative's or senator's district/state office locations (not the Washington DC office).

Start by examining the given page content. Look for links to:
- "Contact" pages
- "Offices" or "District Offices" pages
- "Locations" pages
- Footer links with office information

Use the fetch_page tool to navigate to promising URLs. When you find a page that contains actual office addresses (street addresses, cities, phone numbers), return that URL as your final answer.

Important:
- Return ONLY the URL where you found office addresses, nothing else
- If you can't find office addresses after reasonable navigation, return "NOT_FOUND"
- Do not navigate to more than 5 pages total`

// NavigateToContact uses Claude Haiku with tool_use to find the contact/offices page
func NavigateToContact(baseURL string, debug bool) (string, error) {
	client := anthropic.NewClient(option.WithAPIKey(anthropicAPIKey))

	// Ensure URL has scheme
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	// Fetch the initial page
	initialContent, err := fetchPageContent(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch initial page: %w", err)
	}

	if debug {
		log.Printf("Fetched initial content from %s (%d chars)", baseURL, len(initialContent))
	}

	// Define the fetch_page tool
	fetchPageTool := anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "fetch_page",
			Description: anthropic.String("Fetches a webpage and returns its text content. Use this to navigate to pages that might contain office addresses."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to fetch. Can be absolute or relative to the current page.",
					},
				},
				Required: []string{"url"},
			},
		},
	}

	// Start the conversation
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(
			fmt.Sprintf("I'm starting at this page: %s\n\nPage content:\n%s\n\nPlease find the page with district office addresses.", baseURL, initialContent),
		)),
	}

	currentURL := baseURL
	maxIterations := 6 // Allow up to 6 tool calls

	for i := 0; i < maxIterations; i++ {
		resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.ModelClaude3_5HaikuLatest,
			MaxTokens: 1024,
			System: []anthropic.TextBlockParam{
				{
					Type: "text",
					Text: NavigatorSystemPrompt,
				},
			},
			Tools:    []anthropic.ToolUnionParam{fetchPageTool},
			Messages: messages,
		})
		if err != nil {
			return "", fmt.Errorf("API error: %w", err)
		}

		if debug {
			log.Printf("Response stop reason: %s", resp.StopReason)
		}

		// Check if we got a final text response
		if resp.StopReason == anthropic.StopReasonEndTurn {
			for _, block := range resp.Content {
				if block.Type == "text" {
					result := strings.TrimSpace(block.Text)
					if debug {
						log.Printf("Final response text: %s", result)
					}
					if strings.Contains(result, "NOT_FOUND") {
						return "", fmt.Errorf("could not find office addresses on %s", baseURL)
					}
					// Try to extract URL from the response
					extractedURL := extractURLFromText(result)
					if extractedURL != "" {
						// Make absolute if needed
						if strings.HasPrefix(extractedURL, "/") {
							parsed, _ := url.Parse(currentURL)
							return fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, extractedURL), nil
						}
						return extractedURL, nil
					}
				}
			}
			return "", fmt.Errorf("no valid URL in response")
		}

		// Check for tool use
		if resp.StopReason == anthropic.StopReasonToolUse {
			// Add assistant's response to messages
			messages = append(messages, resp.ToParam())

			// Process each tool use
			var toolResults []anthropic.ContentBlockParamUnion
			for _, block := range resp.Content {
				if block.Type == "tool_use" {
					if block.Name == "fetch_page" {
						// Extract URL from input
						var input struct {
							URL string `json:"url"`
						}
						if err := json.Unmarshal(block.Input, &input); err != nil {
							toolResults = append(toolResults, anthropic.NewToolResultBlock(
								block.ID,
								fmt.Sprintf("Error parsing input: %s", err),
								true,
							))
							continue
						}

						fetchURL := input.URL

						// Make URL absolute if relative
						fetchURL = makeAbsoluteURL(currentURL, fetchURL)
						currentURL = fetchURL

						if debug {
							log.Printf("Fetching page: %s", fetchURL)
						}

						content, err := fetchPageContent(fetchURL)
						if err != nil {
							toolResults = append(toolResults, anthropic.NewToolResultBlock(
								block.ID,
								fmt.Sprintf("Error fetching page: %s", err),
								true,
							))
						} else {
							// Truncate if too long
							if len(content) > 50000 {
								content = content[:50000] + "\n... [truncated]"
							}
							toolResults = append(toolResults, anthropic.NewToolResultBlock(
								block.ID,
								fmt.Sprintf("Page content from %s:\n%s", fetchURL, content),
								false,
							))
						}
					}
				}
			}

			messages = append(messages, anthropic.NewUserMessage(toolResults...))
		}
	}

	return "", fmt.Errorf("max iterations reached without finding office page")
}

// fetchPageContent fetches a URL and converts HTML to text
func fetchPageContent(contentURL string) (string, error) {
	_, err := url.ParseRequestURI(contentURL)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(contentURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status code %d for url %s", resp.StatusCode, contentURL)
	}

	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	text, err := html2text.FromString(string(html), html2text.Options{TextOnly: true})
	if err != nil {
		return "", fmt.Errorf("can't parse html to string: %s", err)
	}

	return text, nil
}

// makeAbsoluteURL converts a relative URL to absolute based on the current page
func makeAbsoluteURL(currentURL, targetURL string) string {
	if strings.HasPrefix(targetURL, "http://") || strings.HasPrefix(targetURL, "https://") {
		return targetURL
	}

	parsed, err := url.Parse(currentURL)
	if err != nil {
		return targetURL
	}

	if strings.HasPrefix(targetURL, "//") {
		return parsed.Scheme + ":" + targetURL
	}

	if strings.HasPrefix(targetURL, "/") {
		return fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, targetURL)
	}

	// Relative path
	basePath := parsed.Path
	if lastSlash := strings.LastIndex(basePath, "/"); lastSlash >= 0 {
		basePath = basePath[:lastSlash+1]
	}
	return fmt.Sprintf("%s://%s%s%s", parsed.Scheme, parsed.Host, basePath, targetURL)
}

// extractURLFromText extracts a URL from text that might contain other content
func extractURLFromText(text string) string {
	// First try exact match (text is just a URL)
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		// Remove any trailing punctuation
		trimmed = strings.TrimRight(trimmed, ".,;:!?")
		if !strings.Contains(trimmed, " ") && !strings.Contains(trimmed, "\n") {
			return trimmed
		}
	}

	// Use regex to find URLs in the text
	matches := urlRegex.FindAllString(text, -1)
	if len(matches) > 0 {
		// Return the last URL found (usually the answer)
		result := matches[len(matches)-1]
		// Clean up trailing punctuation
		result = strings.TrimRight(result, ".,;:!?")
		return result
	}

	return ""
}
