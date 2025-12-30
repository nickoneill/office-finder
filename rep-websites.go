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
	Name struct {
		First        string `yaml:"first"`
		Last         string `yaml:"last"`
		OfficialFull string `yaml:"official_full"`
	} `yaml:"name"`
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

// LegislatorInfo contains display information for a legislator
type LegislatorInfo struct {
	Name  string // Official full name
	State string // Two-letter state code
	Title string // "Rep." or "Sen."
}

// lookupLegislatorInfo returns display info for a bioguide ID
func lookupLegislatorInfo(bioguide string) (*LegislatorInfo, error) {
	url := "https://raw.githubusercontent.com/unitedstates/congress-legislators/main/legislators-current.yaml"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var legislators []Legislator
	if err := yaml.Unmarshal(body, &legislators); err != nil {
		return nil, err
	}

	for _, leg := range legislators {
		if leg.ID.Bioguide == bioguide {
			info := &LegislatorInfo{}

			if leg.Name.OfficialFull != "" {
				info.Name = leg.Name.OfficialFull
			} else {
				info.Name = leg.Name.First + " " + leg.Name.Last
			}

			// Get state and type from latest term
			if len(leg.Terms) > 0 {
				latest := leg.Terms[len(leg.Terms)-1]
				info.State = latest.State
				if latest.Type == "sen" {
					info.Title = "Sen."
				} else {
					info.Title = "Rep."
				}
			}

			return info, nil
		}
	}

	return nil, nil
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
