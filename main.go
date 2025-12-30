package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

// anthropicAPIKey is loaded from environment
var anthropicAPIKey string

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

// NavigateResult is the JSON output of the navigate command
type NavigateResult struct {
	ContactURL string `json:"contact_url"`
}

func main() {
	anthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")

	app := &cli.App{
		Name:  "office-finder",
		Usage: "A tool to find and extract representative office addresses using Claude AI",
		Commands: []*cli.Command{
			{
				Name:      "navigate",
				Usage:     "Find the contact/offices page URL for a representative website",
				ArgsUsage: "<url>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "debug",
						Usage: "Enable debug mode",
						Value: false,
					},
				},
				Action: func(ctx *cli.Context) error {
					if anthropicAPIKey == "" {
						return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
					}
					if ctx.NArg() < 1 {
						return fmt.Errorf("url argument is required")
					}
					url := ctx.Args().First()
					debug := ctx.Bool("debug")

					contactURL, err := NavigateToContact(url, debug)
					if err != nil {
						return err
					}

					result := NavigateResult{ContactURL: contactURL}
					output, _ := json.Marshal(result)
					fmt.Println(string(output))
					return nil
				},
			},
			{
				Name:  "extract",
				Usage: "Extract office information from a specific page",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "url",
						Usage:    "URL of the page containing office addresses",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "bioguide",
						Usage:    "Bioguide ID of the legislator",
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "debug",
						Usage: "Enable debug mode",
						Value: false,
					},
				},
				Action: func(ctx *cli.Context) error {
					if anthropicAPIKey == "" {
						return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
					}
					url := ctx.String("url")
					bioguide := ctx.String("bioguide")
					debug := ctx.Bool("debug")

					offices, err := ExtractOffices(url, bioguide, debug)
					if err != nil {
						return err
					}

					// Output as YAML in district-offices format
					output := YAMLLegislatorOffices{
						Offices: offices,
					}
					output.ID.Bioguide = bioguide

					yamlOutput, err := yaml.Marshal([]YAMLLegislatorOffices{output})
					if err != nil {
						return err
					}
					fmt.Println(string(yamlOutput))
					return nil
				},
			},
			{
				Name:  "batch",
				Usage: "Process all legislators in batch mode",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "state",
						Usage: "Filter by state (e.g., CA, TX)",
					},
					&cli.StringFlag{
						Name:  "bioguide",
						Usage: "Process only a specific bioguide ID",
					},
					&cli.BoolFlag{
						Name:  "debug",
						Usage: "Enable debug mode",
						Value: false,
					},
				},
				Action: func(ctx *cli.Context) error {
					if anthropicAPIKey == "" {
						return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
					}
					state := ctx.String("state")
					bioguide := ctx.String("bioguide")
					debug := ctx.Bool("debug")
					return runBatch(state, bioguide, debug)
				},
			},
			{
				Name:  "compare",
				Usage: "Compare results with upstream district-offices.yaml",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Show detailed changes for each legislator",
						Value:   false,
					},
				},
				Action: func(ctx *cli.Context) error {
					verbose := ctx.Bool("verbose")
					return runCompare(verbose)
				},
			},
			{
				Name:  "lintYAML",
				Usage: "Re-sort the yaml file and fill in other IDs",
				Action: func(ctx *cli.Context) error {
					return lintYAML()
				},
			},
			{
				Name:  "merge",
				Usage: "Merge a result file into legislators-district-offices.yaml",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "bioguide",
						Usage:    "Bioguide ID to merge",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "target",
						Usage:    "Path to legislators-district-offices.yaml",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "results",
						Usage: "Path to results directory",
						Value: "results",
					},
				},
				Action: func(ctx *cli.Context) error {
					bioguide := ctx.String("bioguide")
					target := ctx.String("target")
					resultsDir := ctx.String("results")
					return runMerge(bioguide, target, resultsDir)
				},
			},
			{
				Name:      "name",
				Usage:     "Look up a legislator's display info by bioguide ID",
				ArgsUsage: "<bioguide>",
				Action: func(ctx *cli.Context) error {
					if ctx.NArg() < 1 {
						return fmt.Errorf("bioguide ID required")
					}
					bioguide := ctx.Args().Get(0)
					info, err := lookupLegislatorInfo(bioguide)
					if err != nil {
						return err
					}
					if info == nil {
						return fmt.Errorf("legislator %s not found", bioguide)
					}
					// Output format: "CA Rep. Pete Aguilar"
					fmt.Printf("%s %s %s\n", info.State, info.Title, info.Name)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
