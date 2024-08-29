package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/sashabaranov/go-openai"
	"github.com/urfave/cli/v2"
)

type OfficeList struct {
	URL     string       `json:"url"`
	Offices []OfficeInfo `json:"offices"`
}

type OfficeInfo struct {
	Address  string `json:"address"`
	Suite    string `json:"suite,omitempty"`
	Building string `json:"building,omitempty"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Phone    string `json:"phone"`
	Fax      string `json:"fax"`
}

var openaiClient *openai.Client

func main() {
	app := &cli.App{
		Name:  "office-finder",
		Usage: "A tool to scrape and process representative office addresses and phone numbers",
		Commands: []*cli.Command{
			{
				Name:  "scrape",
				Usage: "Scrape office addresses from public representative websites",
				Action: func(ctx *cli.Context) error {
					return scrapeAllURLs()
				},
			},
			{
				Name:  "validate",
				Usage: "Validate legislators in offices.json against the YAML file",
				Action: func(ctx *cli.Context) error {
					return validateLegislators()
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func validateLegislators() error {
	repURLs := listRepURLs()

	// Read offices.json
	officesData, err := os.ReadFile("offices.json")
	if err != nil {
		return fmt.Errorf("error reading offices.json: %v", err)
	}

	var officeList []OfficeList
	err = json.Unmarshal(officesData, &officeList)
	if err != nil {
		return fmt.Errorf("error parsing offices.json: %v", err)
	}

	existingOffices := map[string]bool{}
	for _, office := range officeList {
		if len(office.Offices) > 0 {
			existingOffices[office.URL] = true
		}
	}

	for _, repURL := range repURLs {
		if !existingOffices[repURL] {
			log.Printf("didn't find offices for %s", repURL)
		}
	}

	return nil
}

func fetchYAMLData() ([]byte, error) {
	url := "https://raw.githubusercontent.com/unitedstates/congress-legislators/main/legislators-current.yaml"

	// Download the YAML file
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	return body, nil
}
