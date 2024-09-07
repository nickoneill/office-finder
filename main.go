package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/sashabaranov/go-openai"
	"github.com/urfave/cli/v2"
)

type OfficeList struct {
	Bioguide string       `json:"bioguide"`
	URL      string       `json:"url"`
	Offices  []OfficeInfo `json:"offices"`
}

type OfficeInfo struct {
	Address  string `json:"address"`
	Suite    string `json:"suite,omitempty"`
	Building string `json:"building,omitempty"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Phone    string `json:"phone,omitempty"`
	Fax      string `json:"fax,omitempty"`
}

var openaiClient *openai.Client

func main() {
	openaiToken := os.Getenv("OPENAI_API_KEY")
	if openaiToken == "" {
		log.Fatal("no OpenAI token found")
	}
	openaiClient = openai.NewClient(openaiToken)

	app := &cli.App{
		Name:  "office-finder",
		Usage: "A tool to scrape and process representative office addresses and phone numbers",
		Commands: []*cli.Command{
			{
				Name:  "scrape",
				Usage: "Scrape office addresses from public representative websites",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "url",
						Usage: "URL to scrape (optional, if not provided all URLs will be scraped)",
					},
					&cli.BoolFlag{
						Name:  "debug",
						Usage: "Enable debug mode",
						Value: false,
					},
				},
				Action: func(ctx *cli.Context) error {
					url := ctx.String("url")
					debug := ctx.Bool("debug")
					if url == "" {
						return scrapeAllURLs()
					}
					return scrapeOne(url, debug)
				},
			},
			{
				Name:  "validate",
				Usage: "Validate legislators in offices.json against the YAML file",
				Action: func(ctx *cli.Context) error {
					return validateLegislators()
				},
			},
			{
				Name:  "upstreamChanges",
				Usage: "Update the YAML file with office information from offices.json",
				Action: func(ctx *cli.Context) error {
					return upstreamChanges()
				},
			},
			{
				Name:  "lintYAML",
				Usage: "Re-sort the yaml file and fill in other IDs",
				Action: func(ctx *cli.Context) error {
					return lintYAML()
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

	// TODO: ensure offices listed are in the state they're supposed to be
	// TODO: ensure states are two letter abbreviations

	return nil
}
