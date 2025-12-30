package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// runBatch processes legislators in batch mode
func runBatch(stateFilter, bioguideFilter string, debug bool) error {
	// Fetch legislators
	legislators, err := fetchLegislators()
	if err != nil {
		return fmt.Errorf("failed to fetch legislators: %w", err)
	}

	// Load cache
	cache, err := LoadCache()
	if err != nil {
		log.Printf("Warning: failed to load cache: %v", err)
		cache = make(URLCache)
	}

	// Ensure results directory exists
	if err := os.MkdirAll("results", 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	// Filter legislators
	var toProcess []Legislator
	for _, leg := range legislators {
		if len(leg.Terms) == 0 {
			continue
		}
		latestTerm := leg.Terms[len(leg.Terms)-1]

		// Check if still current
		endDate, err := time.Parse("2006-01-02", latestTerm.End)
		if err != nil || endDate.Before(time.Now()) {
			continue
		}

		// Apply filters
		if bioguideFilter != "" && leg.ID.Bioguide != bioguideFilter {
			continue
		}
		if stateFilter != "" && !strings.EqualFold(latestTerm.State, stateFilter) {
			continue
		}
		if latestTerm.URL == "" {
			continue
		}

		toProcess = append(toProcess, leg)
	}

	log.Printf("Processing %d legislators", len(toProcess))

	// Process legislators with rate limiting
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Max 3 concurrent
	rateLimiter := time.Tick(2 * time.Second)

	for _, leg := range toProcess {
		wg.Add(1)
		go func(legislator Legislator) {
			defer wg.Done()
			semaphore <- struct{}{}
			<-rateLimiter
			defer func() { <-semaphore }()

			bioguide := legislator.ID.Bioguide
			websiteURL := legislator.Terms[len(legislator.Terms)-1].URL

			log.Printf("Processing %s (%s)", bioguide, websiteURL)

			// Check cache for contact URL
			contactURL, cached := cache.GetCachedURL(bioguide)
			if !cached {
				// Navigate to find contact URL
				var err error
				contactURL, err = NavigateToContact(websiteURL, debug)
				if err != nil {
					log.Printf("Error navigating for %s: %v", bioguide, err)
					return
				}
				cache.SetCachedURL(bioguide, contactURL)
			} else if debug {
				log.Printf("Using cached URL for %s: %s", bioguide, contactURL)
			}

			// Extract offices
			offices, err := ExtractOffices(contactURL, bioguide, debug)
			if err != nil {
				log.Printf("Error extracting offices for %s: %v", bioguide, err)
				return
			}

			// Save result
			result := YAMLLegislatorOffices{
				Offices: offices,
			}
			result.ID.Bioguide = bioguide
			result.ID.Govtrack = legislator.ID.Govtrack

			yamlOutput, err := yaml.Marshal([]YAMLLegislatorOffices{result})
			if err != nil {
				log.Printf("Error marshaling YAML for %s: %v", bioguide, err)
				return
			}

			resultFile := fmt.Sprintf("results/%s.yaml", bioguide)
			if err := os.WriteFile(resultFile, yamlOutput, 0644); err != nil {
				log.Printf("Error writing result for %s: %v", bioguide, err)
				return
			}

			log.Printf("Saved %d offices for %s", len(offices), bioguide)
		}(leg)
	}

	wg.Wait()

	// Save updated cache
	if err := SaveCache(cache); err != nil {
		log.Printf("Warning: failed to save cache: %v", err)
	}

	log.Printf("Batch processing complete")
	return nil
}

// fetchLegislators downloads and parses the legislators-current.yaml file
func fetchLegislators() ([]Legislator, error) {
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

	return legislators, nil
}
