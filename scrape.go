package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"jaytaylor.com/html2text"
)

const ADDRESS_PROMPT = `please find all office addresses within this content, returning them in json formatting as plain text without any backticks or formatting indicators. Include the fields: address, city, state, zip, phone.
Format all phone numbers as 123-456-7890
If a fax number is listed, also include it in a fax field.
If the address includes a suite number or room, also include it in a suite field.
if the address includes a building, also include it in a building field.`

const LOCATIONS_PROMPT = `please return only the most likely url on this page that would list office locations without any other text`

func scrapeAllURLs() error {
	openaiToken := os.Getenv("OPENAI_API_KEY")
	if openaiToken == "" {
		return fmt.Errorf("no OpenAI token found")
	}
	openaiClient = openai.NewClient(openaiToken)

	bioguideToURLs := listRepURLs()
	log.Printf("got %d urls", len(bioguideToURLs))

	results := processURLs(bioguideToURLs)
	// sort results by bioguide for consistent diffs
	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Bioguide) < strings.ToLower(results[j].Bioguide)
	})

	file, err := os.Create("offices.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(results)
	if err != nil {
		return err
	}

	return nil
}

func processURLs(urls map[string]string) []OfficeList {
	var results []OfficeList

	// this could certainly be faster done in parallel but in an effort to not run afoul of
	// openai rate limits we'll do it serially to start
	for bioguide, url := range urls {
		offices, err := findAddresses(url)
		if err != nil {
			log.Printf("Error processing %s: %v", url, err)
			continue
		}
		results = append(results, OfficeList{Bioguide: bioguide, URL: url, Offices: offices})
	}

	return results
}

func getPageSource(contentURL string) (string, error) {
	_, err := url.ParseRequestURI(contentURL)
	if err != nil {
		return "", err
	}

	res, err := http.Get(contentURL)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("status code %d for url %s", res.StatusCode, contentURL)
	}

	html, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}

	return string(html), nil
}

func findAddresses(contentURL string) ([]OfficeInfo, error) {
	log.Printf("finding for %s", contentURL)
	var offices []OfficeInfo

	html, err := getPageSource(contentURL)
	if err != nil {
		return offices, err
	}

	htmlText, err := html2text.FromString(string(html), html2text.Options{TextOnly: true})
	if err != nil {
		return offices, fmt.Errorf("can't parse html to string: %s", err)
	}

	addressResponse, err := getOpenAIResponse(ADDRESS_PROMPT, htmlText, true)
	if err != nil {
		return offices, err
	}
	log.Printf("address response: %s", addressResponse)
	offices, err = marshalOffliceList(addressResponse)
	if err != nil {
		return offices, err
	}

	if len(offices) == 0 {
		log.Printf("couldn't get office locations at %s", contentURL)
		// see if we can get a better url
		locationsURL, err := getOpenAIResponse(LOCATIONS_PROMPT, string(html), false)
		if err != nil {
			return offices, err
		}

		log.Printf("trying alternative for %s, %s", contentURL, locationsURL)
		html, err = getPageSource(locationsURL)
		if err != nil {
			return offices, err
		}

		htmlText, err := html2text.FromString(string(html), html2text.Options{TextOnly: true})
		if err != nil {
			return offices, fmt.Errorf("can't parse html to string: %s", err)
		}

		addressResponse, err := getOpenAIResponse(ADDRESS_PROMPT, htmlText, true)
		if err != nil {
			return offices, err
		}
		log.Printf("address response: %s", addressResponse)

		offices, err := marshalOffliceList(addressResponse)

		return offices, err
	}

	return offices, err
}

func getOpenAIResponse(prompt, content string, structuredOutput bool) (string, error) {
	request := openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: prompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: content,
			},
		},
	}

	// when asking for json formatted information, providing a schema makes the resulting data much
	// more reliable without having to add too much extra prompt text
	if structuredOutput {
		request.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Strict: true,
				Name:   "address_response",
				Schema: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"addresses": {
							Type: jsonschema.Array,
							Items: &jsonschema.Definition{
								Type: jsonschema.Object,
								Properties: map[string]jsonschema.Definition{
									"address": {
										Type:        jsonschema.String,
										Description: "The street address of the office",
									},
									"city": {
										Type:        jsonschema.String,
										Description: "The city where the office is located",
									},
									"state": {
										Type:        jsonschema.String,
										Description: "The state where the office is located",
									},
									"zip": {
										Type:        jsonschema.String,
										Description: "The ZIP code of the office",
									},
									"phone": {
										Type:        jsonschema.String,
										Description: "The phone number of the office",
									},
									"fax": {
										Type:        jsonschema.String,
										Description: "The fax number of the office",
									},
									"suite": {
										Type:        jsonschema.String,
										Description: "The suite number or floor of the office",
									},
									"building": {
										Type:        jsonschema.String,
										Description: "The building that the office is in",
									},
								},
								Required:             []string{"address", "city", "state", "zip", "phone", "fax", "suite", "building"},
								AdditionalProperties: false,
							},
						},
					},
					Required:             []string{"addresses"},
					AdditionalProperties: false,
				},
			},
		}
	}

	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		request,
	)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

type OpenAIOfficeResponse struct {
	Offices []OfficeInfo `json:"addresses"`
}

func marshalOffliceList(officeList string) ([]OfficeInfo, error) {
	var offices OpenAIOfficeResponse
	err := json.Unmarshal([]byte(officeList), &offices)
	if err != nil {
		return offices.Offices, err
	}

	return offices.Offices, nil
}
