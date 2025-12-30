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

// runMerge merges a result file into the target legislators-district-offices.yaml
func runMerge(bioguide string, targetFile string, resultsDir string) error {
	// Read the result file
	resultPath := fmt.Sprintf("%s/%s.yaml", resultsDir, bioguide)
	resultData, err := os.ReadFile(resultPath)
	if err != nil {
		return fmt.Errorf("failed to read result file %s: %w", resultPath, err)
	}

	var resultLegislators []YAMLLegislatorOffices
	if err := yaml.Unmarshal(resultData, &resultLegislators); err != nil {
		return fmt.Errorf("failed to parse result file: %w", err)
	}

	if len(resultLegislators) == 0 {
		return fmt.Errorf("no data in result file for %s", bioguide)
	}

	result := resultLegislators[0]

	// Read the target file
	targetData, err := os.ReadFile(targetFile)
	if err != nil {
		return fmt.Errorf("failed to read target file %s: %w", targetFile, err)
	}

	var legislators []YAMLLegislatorOffices
	if err := yaml.Unmarshal(targetData, &legislators); err != nil {
		return fmt.Errorf("failed to parse target file: %w", err)
	}

	// Find and replace, or add new entry
	found := false
	for i, leg := range legislators {
		if leg.ID.Bioguide == bioguide {
			// Preserve govtrack and thomas IDs from existing entry
			result.ID.Govtrack = leg.ID.Govtrack
			result.ID.Thomas = leg.ID.Thomas

			// Preserve lat/lng and other fields from matching offices
			preserveOfficeFields(leg.Offices, result.Offices)

			legislators[i] = result
			found = true
			log.Printf("Updated existing entry for %s", bioguide)
			break
		}
	}

	if !found {
		// Add new entry - need to fetch IDs from legislators-current.yaml
		if err := fillMissingIDs(&result); err != nil {
			log.Printf("Warning: could not fill IDs for %s: %v", bioguide, err)
		}
		legislators = append(legislators, result)
		log.Printf("Added new entry for %s", bioguide)
	}

	// Sort by bioguide
	sort.Slice(legislators, func(i, j int) bool {
		return strings.ToLower(legislators[i].ID.Bioguide) < strings.ToLower(legislators[j].ID.Bioguide)
	})

	// Marshal back to YAML
	output, err := yaml.Marshal(legislators)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Convert double quotes to single quotes for consistency with upstream style
	singleQuotedOutput := strings.ReplaceAll(string(output), `"`, `'`)

	// Write back to target file
	if err := os.WriteFile(targetFile, []byte(singleQuotedOutput), 0644); err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	log.Printf("Successfully merged %s into %s", bioguide, targetFile)
	return nil
}

// preserveOfficeFields copies lat/lng and other fields from existing offices to new offices
// when the office address matches. Also preserves the original address/suite/city formatting
// to avoid unnecessary diffs.
func preserveOfficeFields(existing []YAMLOffice, updated []YAMLOffice) {
	for i := range updated {
		for _, existingOffice := range existing {
			// Use the same matching logic as compare.go
			if officeEquals(existingOffice, OfficeInfo{
				Address: updated[i].Address,
				Suite:   updated[i].Suite,
				City:    updated[i].City,
			}) {
				// Preserve original address formatting to avoid unnecessary diffs
				updated[i].Address = existingOffice.Address
				updated[i].Suite = existingOffice.Suite
				updated[i].City = existingOffice.City
				updated[i].ID = existingOffice.ID

				// Preserve fields that we don't scrape
				if updated[i].Latitude == 0 && existingOffice.Latitude != 0 {
					updated[i].Latitude = existingOffice.Latitude
				}
				if updated[i].Longitude == 0 && existingOffice.Longitude != 0 {
					updated[i].Longitude = existingOffice.Longitude
				}
				if updated[i].Hours == "" && existingOffice.Hours != "" {
					updated[i].Hours = existingOffice.Hours
				}
				if updated[i].Fax == "" && existingOffice.Fax != "" {
					updated[i].Fax = existingOffice.Fax
				}
				break
			}
		}
	}
}

// fillMissingIDs fetches govtrack and thomas IDs for a new legislator
func fillMissingIDs(leg *YAMLLegislatorOffices) error {
	resp, err := http.Get("https://raw.githubusercontent.com/unitedstates/congress-legislators/main/legislators-current.yaml")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var legislators []Legislator
	if err := yaml.Unmarshal(body, &legislators); err != nil {
		return err
	}

	for _, l := range legislators {
		if l.ID.Bioguide == leg.ID.Bioguide {
			leg.ID.Govtrack = l.ID.Govtrack
			leg.ID.Thomas = l.ID.Thomas
			return nil
		}
	}

	return fmt.Errorf("legislator %s not found in legislators-current.yaml", leg.ID.Bioguide)
}
