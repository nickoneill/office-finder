package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const ExtractorSystemPrompt = `You are extracting office address information from a congressional website.

Extract ALL district/state offices from the page content. For each office, provide:
- address: The street address (do NOT include suite, room, or building info here)
- suite: Suite, room, or floor number if present (e.g., "Suite 100", "Room 205")
- building: Building name if mentioned (e.g., "Federal Building")
- city: The city name
- state: The two-letter state abbreviation (e.g., "CA", "TX")
- zip: The ZIP code (5 digits)
- phone: The phone number
- fax: The fax number if available

IMPORTANT:
- Skip Washington DC offices (these are the Capitol offices, not district offices)
- Return an empty array if no offices are found
- Ensure state is always a two-letter abbreviation
- Phone numbers should include area code

Return the data as a JSON object with an "offices" array.`

type ExtractorResponse struct {
	Offices []OfficeInfo `json:"offices"`
}

// ExtractOffices uses Claude Sonnet to extract office data from a specific page
func ExtractOffices(pageURL string, bioguide string, debug bool) ([]YAMLOffice, error) {
	client := anthropic.NewClient(option.WithAPIKey(anthropicAPIKey))

	// Fetch the page content
	content, err := fetchPageContent(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	if debug {
		log.Printf("Fetched content from %s (%d chars)", pageURL, len(content))
	}

	// Call Claude Sonnet for extraction
	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_0,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: ExtractorSystemPrompt,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				fmt.Sprintf("Please extract all district office information from this page:\n\n%s", content),
			)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("API error: %w", err)
	}

	// Extract JSON from response
	var responseText string
	for _, block := range resp.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	if debug {
		log.Printf("Raw response: %s", responseText)
	}

	// Parse the JSON - handle both raw JSON and markdown code blocks
	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var extractedData ExtractorResponse
	if err := json.Unmarshal([]byte(jsonStr), &extractedData); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	// Convert to YAML format
	var offices []YAMLOffice
	for _, office := range extractedData.Offices {
		// Skip DC offices
		if strings.ToLower(office.City) == "washington" ||
			strings.ToLower(office.State) == "d.c." ||
			strings.ToLower(office.State) == "dc" {
			continue
		}

		yamlOffice := YAMLOffice{
			ID:       nextOfficeKey(bioguide, office.City, offices),
			Address:  office.Address,
			Suite:    formatSuite(office.Suite),
			Building: office.Building,
			City:     office.City,
			State:    formatState(office.State),
			Zip:      office.Zip,
			Phone:    formatPhone(office.Phone),
			Fax:      formatPhone(office.Fax),
		}
		offices = append(offices, yamlOffice)
	}

	return offices, nil
}

// extractJSON extracts JSON from a response that might be wrapped in markdown code blocks
func extractJSON(text string) string {
	text = strings.TrimSpace(text)

	// Try to find JSON in markdown code blocks
	if idx := strings.Index(text, "```json"); idx >= 0 {
		start := idx + 7
		if endIdx := strings.Index(text[start:], "```"); endIdx >= 0 {
			return strings.TrimSpace(text[start : start+endIdx])
		}
	}

	// Try to find JSON in generic code blocks
	if idx := strings.Index(text, "```"); idx >= 0 {
		start := idx + 3
		// Skip any language identifier
		if nlIdx := strings.Index(text[start:], "\n"); nlIdx >= 0 {
			start = start + nlIdx + 1
		}
		if endIdx := strings.Index(text[start:], "```"); endIdx >= 0 {
			return strings.TrimSpace(text[start : start+endIdx])
		}
	}

	// Try to find raw JSON object
	if idx := strings.Index(text, "{"); idx >= 0 {
		// Find matching closing brace
		depth := 0
		for i := idx; i < len(text); i++ {
			switch text[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return text[idx : i+1]
				}
			}
		}
	}

	return ""
}
