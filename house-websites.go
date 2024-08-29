package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"gopkg.in/yaml.v2"
)

type Legislator struct {
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

func listRepURLs() []string {
	url := "https://raw.githubusercontent.com/unitedstates/congress-legislators/main/legislators-current.yaml"

	websiteURLs := []string{}

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
				websiteURLs = append(websiteURLs, latestTerm.URL)
			}
		}
	}

	return websiteURLs
}