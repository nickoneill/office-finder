package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

const UpdatedYAMLFile = "updated_legislators-district-offices.yaml"

// lintYAML expects an updated_legislators-district-offices.yaml file in the current directory,
// sorting it so the legislators appear by bioguide and ensuring other IDs are set
func lintYAML() error {
	yamlFile, err := os.ReadFile(UpdatedYAMLFile)
	if err != nil {
		return fmt.Errorf("error reading YAML file: %v", err)
	}

	var fileLegislators []YAMLLegislatorOffices
	err = yaml.Unmarshal(yamlFile, &fileLegislators)
	if err != nil {
		return fmt.Errorf("error parsing YAML data: %v", err)
	}

	url := "https://raw.githubusercontent.com/unitedstates/congress-legislators/main/legislators-current.yaml"

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error downloading file: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v\n", err)
		return err
	}

	// Parse the YAML
	var legislators []Legislator
	err = yaml.Unmarshal(body, &legislators)
	if err != nil {
		log.Printf("Error parsing YAML: %v\n", err)
		return err
	}

	// sort everyone by bioguide again
	sort.Slice(fileLegislators, func(i, j int) bool {
		return strings.ToLower(fileLegislators[i].ID.Bioguide) < strings.ToLower(fileLegislators[j].ID.Bioguide)
	})

	// set any IDs that are missing
	for i, _ := range fileLegislators {
		for _, leg := range legislators {
			if fileLegislators[i].ID.Bioguide == leg.ID.Bioguide {
				fileLegislators[i].ID.Govtrack = leg.ID.Govtrack
				fileLegislators[i].ID.Thomas = leg.ID.Thomas
			}
		}
	}

	updatedYAML, err := yaml.Marshal(fileLegislators)
	if err != nil {
		return fmt.Errorf("error marshaling updated YAML data: %v", err)
	}

	// we want single quoted strings for zips and numeric IDs so replace them all here
	singleQuotedUpdatedYAML := strings.ReplaceAll(string(updatedYAML), `"`, `'`)

	err = os.WriteFile(UpdatedYAMLFile, []byte(singleQuotedUpdatedYAML), 0644)
	if err != nil {
		return fmt.Errorf("error writing updated YAML file: %v", err)
	}

	log.Printf("done linting")

	return nil
}
