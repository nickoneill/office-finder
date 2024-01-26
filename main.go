package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/sashabaranov/go-openai"
	"jaytaylor.com/html2text"
)

func main() {
	res, err := findAddresses("https://www.lujan.senate.gov")
	if err != nil {
		log.Fatalf("some error: %s", err)
	}

	log.Println(res)
}

func findAddresses(contentURL string) (string, error) {
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

	// readability will try to find the main content of a url, it's
	// not perfect but it does a good enough job for the summarize request
	// doc, err := readability.NewDocument(string(html))
	// if err != nil {
	// 	return "", err
	// }

	// content := doc.Content()
	// log.Printf("content was: %s", content)
	// // readability leaves HTML in place so we need to strip all tags too
	// p := bluemonday.StripTagsPolicy()
	// nonhtml := p.Sanitize(content)

	htmlText, err := html2text.FromString(string(html), html2text.Options{TextOnly: true})
	if err != nil {
		return "", fmt.Errorf("can't parse html to string: %s", err)
	}

	// log.Printf("non-html content: %s", htmlText)
	// return "", fmt.Errorf("error")

	openaiToken := os.Getenv("OPENAI_API_KEY")
	if openaiToken == "" {
		return "", fmt.Errorf("no openai token found")
	}

	client := openai.NewClient(openaiToken)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4TurboPreview,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: `you are an assistant designed to extract addresses from text content, returning them in a yaml format which includes the fields: address, city, state, zip, phone. If a fax number is listed, also include it in a fax field. If the address includes a suite number or room, also include it in a suite field. if the address includes a building, also include it in a building field.`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: addressPrompt(htmlText),
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func addressPrompt(content string) string {
	prompt := fmt.Sprintf("find the addresses in this content: %s", content)
	// log.Println(prompt)

	return prompt
}
