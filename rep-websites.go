package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
)

type Legislator struct {
	ID struct {
		Bioguide string `yaml:"bioguide"`
		Govtrack int    `yaml:"govtrack"`
		Thomas   string `yaml:"thomas,omitempty"`
	} `yaml:"id"`
	Terms []struct {
		Type         string `yaml:"type"`
		Start        string `yaml:"start"`
		End          string `yaml:"end"`
		State        string `yaml:"state"`
		Party        string `yaml:"party"`
		URL          string `yaml:"url"`
		ClassAtStart string `yaml:"class"`
	} `yaml:"terms"`
}

// listRepURLs returns a map of bioguide IDs to website urls
func listRepURLs() map[string]string {
	url := "https://raw.githubusercontent.com/unitedstates/congress-legislators/main/legislators-current.yaml"

	websiteURLs := map[string]string{}

	// Download the YAML file
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error downloading file: %v\n", err)
		return websiteURLs
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v\n", err)
		return websiteURLs
	}

	// Parse the YAML
	var legislators []Legislator
	err = yaml.Unmarshal(body, &legislators)
	if err != nil {
		log.Printf("Error parsing YAML: %v\n", err)
		return websiteURLs
	}

	// Extract URLs of current representatives
	for _, legislator := range legislators {
		if len(legislator.Terms) > 0 {
			latestTerm := legislator.Terms[len(legislator.Terms)-1]
			endDate, err := time.Parse("2006-01-02", latestTerm.End)
			if err != nil {
				log.Printf("Error parsing end date: %v\n", err)
				continue
			}
			if endDate.Before(time.Now()) {
				log.Printf("double checking currency... end date is before now")
				continue
			}
			if latestTerm.Type == "rep" || latestTerm.Type == "sen" {
				websiteURLs[legislator.ID.Bioguide] = latestTerm.URL
			}
		}
	}

	return websiteURLs
}
