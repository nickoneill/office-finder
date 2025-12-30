package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// findEntryBounds finds the start and end byte positions of a legislator entry in raw YAML
func findEntryBounds(data []byte, bioguide string) (start, end int, found bool) {
	// Pattern to find entry starts: "- id:\n    bioguide: XXXXX"
	pattern := regexp.MustCompile(`(?m)^- id:\n    bioguide: ([A-Z]\d+)`)
	matches := pattern.FindAllSubmatchIndex(data, -1)

	for i, match := range matches {
		if len(match) >= 4 {
			matchedBio := string(data[match[2]:match[3]])
			if matchedBio == bioguide {
				start = match[0]
				if i+1 < len(matches) {
					end = matches[i+1][0]
				} else {
					end = len(data)
				}
				return start, end, true
			}
		}
	}
	return 0, 0, false
}

// findInsertPosition finds where to insert a new entry alphabetically
func findInsertPosition(data []byte, bioguide string) int {
	pattern := regexp.MustCompile(`(?m)^- id:\n    bioguide: ([A-Z]\d+)`)
	matches := pattern.FindAllSubmatchIndex(data, -1)

	bioLower := strings.ToLower(bioguide)
	for _, match := range matches {
		if len(match) >= 4 {
			matchedBio := string(data[match[2]:match[3]])
			if strings.ToLower(matchedBio) > bioLower {
				return match[0]
			}
		}
	}
	return len(data)
}

// formatEntry formats a single legislator entry to match upstream style
func formatEntry(leg YAMLLegislatorOffices) []byte {
	var buf bytes.Buffer

	buf.WriteString("- id:\n")
	buf.WriteString(fmt.Sprintf("    bioguide: %s\n", leg.ID.Bioguide))
	buf.WriteString(fmt.Sprintf("    govtrack: %d\n", leg.ID.Govtrack))
	if leg.ID.Thomas != "" {
		buf.WriteString(fmt.Sprintf("    thomas: '%s'\n", leg.ID.Thomas))
	}
	buf.WriteString("  offices:\n")

	for _, o := range leg.Offices {
		buf.WriteString(fmt.Sprintf("  - id: %s\n", o.ID))
		buf.WriteString(fmt.Sprintf("    address: %s\n", o.Address))
		if o.Suite != "" {
			if isNumericString(o.Suite) {
				buf.WriteString(fmt.Sprintf("    suite: '%s'\n", o.Suite))
			} else {
				buf.WriteString(fmt.Sprintf("    suite: %s\n", o.Suite))
			}
		}
		if o.Building != "" {
			buf.WriteString(fmt.Sprintf("    building: %s\n", o.Building))
		}
		buf.WriteString(fmt.Sprintf("    city: %s\n", o.City))
		buf.WriteString(fmt.Sprintf("    state: %s\n", o.State))
		buf.WriteString(fmt.Sprintf("    zip: '%s'\n", o.Zip))
		if o.Hours != "" {
			buf.WriteString(fmt.Sprintf("    hours: %s\n", o.Hours))
		}
		if o.Latitude != 0 {
			buf.WriteString(fmt.Sprintf("    latitude: %v\n", o.Latitude))
		}
		if o.Longitude != 0 {
			buf.WriteString(fmt.Sprintf("    longitude: %v\n", o.Longitude))
		}
		if o.Fax != "" {
			buf.WriteString(fmt.Sprintf("    fax: %s\n", o.Fax))
		}
		if o.Phone != "" {
			buf.WriteString(fmt.Sprintf("    phone: %s\n", o.Phone))
		}
	}

	return buf.Bytes()
}

func isNumericString(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// runMerge surgically replaces only the target legislator's entry in the YAML file
func runMerge(bioguide string, targetFile string, resultsDir string) error {
	// Read and parse the result file
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

	// Read target file as raw bytes
	targetData, err := os.ReadFile(targetFile)
	if err != nil {
		return fmt.Errorf("failed to read target file %s: %w", targetFile, err)
	}

	// Also parse it to get structured data for field preservation
	var legislators []YAMLLegislatorOffices
	if err := yaml.Unmarshal(targetData, &legislators); err != nil {
		return fmt.Errorf("failed to parse target file: %w", err)
	}

	// Find existing entry to preserve fields
	for _, leg := range legislators {
		if leg.ID.Bioguide == bioguide {
			result.ID.Govtrack = leg.ID.Govtrack
			result.ID.Thomas = leg.ID.Thomas
			preserveOfficeFields(leg.Offices, result.Offices)
			break
		}
	}

	// If new entry, fill IDs
	start, end, found := findEntryBounds(targetData, bioguide)
	if !found {
		if err := fillMissingIDs(&result); err != nil {
			log.Printf("Warning: could not fill IDs for %s: %v", bioguide, err)
		}
	}

	// Format the new entry
	newEntry := formatEntry(result)

	// Surgically replace or insert
	var output []byte
	if found {
		output = make([]byte, 0, len(targetData)-end+start+len(newEntry))
		output = append(output, targetData[:start]...)
		output = append(output, newEntry...)
		output = append(output, targetData[end:]...)
		log.Printf("Updated entry for %s", bioguide)
	} else {
		insertPos := findInsertPosition(targetData, bioguide)
		output = make([]byte, 0, len(targetData)+len(newEntry))
		output = append(output, targetData[:insertPos]...)
		output = append(output, newEntry...)
		output = append(output, targetData[insertPos:]...)
		log.Printf("Added new entry for %s", bioguide)
	}

	if err := os.WriteFile(targetFile, output, 0644); err != nil {
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
