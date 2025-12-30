package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

const DEBUG_INFO = false

// sometimes suite numbers contain dots, or letters
var SuiteNumbersRegex = regexp.MustCompile(`^[a-z0-9\.]+$`)

type YAMLLegislatorOffices struct {
	ID struct {
		Bioguide string `yaml:"bioguide"`
		Govtrack int    `yaml:"govtrack"`
		Thomas   string `yaml:"thomas,omitempty"`
	} `yaml:"id"`
	Offices []YAMLOffice `yaml:"offices"`
}

type YAMLOffice struct {
	ID        string  `yaml:"id"`
	Address   string  `yaml:"address"`
	Suite     string  `yaml:"suite,omitempty"`
	Building  string  `yaml:"building,omitempty"`
	City      string  `yaml:"city"`
	State     string  `yaml:"state"`
	Zip       string  `yaml:"zip"`
	Hours     string  `yaml:"hours,omitempty"`
	Latitude  float64 `yaml:"latitude,omitempty"`
	Longitude float64 `yaml:"longitude,omitempty"`
	Fax       string  `yaml:"fax,omitempty"`
	Phone     string  `yaml:"phone,omitempty"`
}

// ChangeEntry represents a change that needs to be made
type ChangeEntry struct {
	Bioguide   string `json:"bioguide"`
	ChangeType string `json:"change_type"` // "added", "modified", "removed"
}

// ChangesOutput is the JSON output of the compare command
type ChangesOutput struct {
	Changes []ChangeEntry `json:"changes"`
	Summary struct {
		Added    int `json:"added"`
		Modified int `json:"modified"`
		Removed  int `json:"removed"`
	} `json:"summary"`
}

// runCompare compares results with upstream and outputs changes.json
func runCompare(verbose bool) error {
	// Fetch upstream district offices
	upstream, err := fetchUpstreamOffices()
	if err != nil {
		return fmt.Errorf("failed to fetch upstream offices: %w", err)
	}

	// Build map for quick lookup
	upstreamMap := make(map[string]YAMLLegislatorOffices)
	for _, leg := range upstream {
		upstreamMap[leg.ID.Bioguide] = leg
	}

	// Read result files
	resultFiles, err := filepath.Glob("results/*.yaml")
	if err != nil {
		return fmt.Errorf("failed to list result files: %w", err)
	}

	if len(resultFiles) == 0 {
		log.Printf("No result files found in results/")
		return nil
	}

	var changes ChangesOutput

	for _, resultFile := range resultFiles {
		data, err := os.ReadFile(resultFile)
		if err != nil {
			log.Printf("Error reading %s: %v", resultFile, err)
			continue
		}

		var resultLegislators []YAMLLegislatorOffices
		if err := yaml.Unmarshal(data, &resultLegislators); err != nil {
			log.Printf("Error parsing %s: %v", resultFile, err)
			continue
		}

		if len(resultLegislators) == 0 {
			continue
		}

		result := resultLegislators[0]
		bioguide := result.ID.Bioguide

		upstreamLeg, exists := upstreamMap[bioguide]
		if !exists {
			// New legislator
			if len(result.Offices) > 0 {
				changes.Changes = append(changes.Changes, ChangeEntry{
					Bioguide:   bioguide,
					ChangeType: "added",
				})
				changes.Summary.Added++
				if verbose {
					log.Printf("[ADDED] %s - new legislator with %d office(s):", bioguide, len(result.Offices))
					for _, office := range result.Offices {
						log.Printf("  + %s, %s", office.Address, office.City)
					}
				}
			}
			continue
		}

		// Compare offices
		if officesChanged(upstreamLeg.Offices, result.Offices) {
			changes.Changes = append(changes.Changes, ChangeEntry{
				Bioguide:   bioguide,
				ChangeType: "modified",
			})
			changes.Summary.Modified++
			if verbose {
				log.Printf("[MODIFIED] %s:", bioguide)
				printOfficeDiff(upstreamLeg.Offices, result.Offices)
			}
		}
	}

	// Output changes.json
	output, err := json.MarshalIndent(changes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}

	if err := os.WriteFile("changes.json", output, 0644); err != nil {
		return fmt.Errorf("failed to write changes.json: %w", err)
	}

	log.Printf("Comparison complete: %d added, %d modified, %d removed",
		changes.Summary.Added, changes.Summary.Modified, changes.Summary.Removed)

	return nil
}

// fetchUpstreamOffices downloads the current district offices from GitHub
func fetchUpstreamOffices() ([]YAMLLegislatorOffices, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/unitedstates/congress-legislators/main/legislators-district-offices.yaml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var legislators []YAMLLegislatorOffices
	if err := yaml.Unmarshal(data, &legislators); err != nil {
		return nil, err
	}

	return legislators, nil
}

// officesChanged checks if the offices have changed
func officesChanged(upstream, result []YAMLOffice) bool {
	if len(upstream) != len(result) {
		return true
	}

	// Check each result office against upstream
	for _, resultOffice := range result {
		found := false
		for _, upstreamOffice := range upstream {
			if officeEquals(upstreamOffice, OfficeInfo{
				Address: resultOffice.Address,
				Suite:   resultOffice.Suite,
				City:    resultOffice.City,
			}) {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}

	return false
}

// printOfficeDiff prints the differences between upstream and result offices
func printOfficeDiff(upstream, result []YAMLOffice) {
	// Find offices in result that aren't in upstream (added)
	for _, resultOffice := range result {
		found := false
		for _, upstreamOffice := range upstream {
			if officeEquals(upstreamOffice, OfficeInfo{
				Address: resultOffice.Address,
				Suite:   resultOffice.Suite,
				City:    resultOffice.City,
			}) {
				found = true
				break
			}
		}
		if !found {
			suiteInfo := ""
			if resultOffice.Suite != "" {
				suiteInfo = fmt.Sprintf(" (%s)", resultOffice.Suite)
			}
			log.Printf("  + %s%s, %s", resultOffice.Address, suiteInfo, resultOffice.City)
		}
	}

	// Find offices in upstream that aren't in result (removed)
	for _, upstreamOffice := range upstream {
		found := false
		for _, resultOffice := range result {
			if officeEquals(upstreamOffice, OfficeInfo{
				Address: resultOffice.Address,
				Suite:   resultOffice.Suite,
				City:    resultOffice.City,
			}) {
				found = true
				break
			}
		}
		if !found {
			suiteInfo := ""
			if upstreamOffice.Suite != "" {
				suiteInfo = fmt.Sprintf(" (%s)", upstreamOffice.Suite)
			}
			log.Printf("  - %s%s, %s", upstreamOffice.Address, suiteInfo, upstreamOffice.City)
		}
	}
}

func cityKey(city string) string {
	// replace spaces or periods with underscores (yes, st__george is the right style for this key)
	return strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(city), " ", "_"), ".", "_")
}

func officeEquals(office YAMLOffice, genOffice OfficeInfo) bool {
	sameAddress := normalizeAddress(office.Address) == normalizeAddress(genOffice.Address) &&
		normalizeCity(office.City) == normalizeCity(genOffice.City) &&
		normalizeSuite(office.Suite) == normalizeSuite(genOffice.Suite)

	if !sameAddress && DEBUG_INFO {
		log.Printf("compared address: %s %s", normalizeAddress(office.Address), normalizeAddress(genOffice.Address))
		log.Printf("compared city: %s %s", normalizeCity(office.City), normalizeCity(genOffice.City))
		log.Printf("compared suite: %s %s", normalizeSuite(office.Suite), normalizeSuite(genOffice.Suite))
	}

	return sameAddress
}

// nextOfficeKey generates subsequent city keys for duplicates like philadelphia-1, philadelphia-2, etc
func nextOfficeKey(bioguide, city string, existingOffices []YAMLOffice) string {
	baseCityKey := fmt.Sprintf("%s-%s", bioguide, cityKey(city))

	cityCount := 0
	for _, office := range existingOffices {
		if strings.HasPrefix(office.ID, baseCityKey) {
			suffix := strings.TrimPrefix(office.ID, baseCityKey)
			if suffix == "" {
				cityCount = 1
				continue
			}
			if suffix[0] == '-' {
				num, err := strconv.Atoi(suffix[1:])
				if err == nil {
					cityCount = num + 1
				}
			}
		}
	}

	if cityCount > 0 {
		return fmt.Sprintf("%s-%d", baseCityKey, cityCount)
	}
	return baseCityKey
}

func formatSuite(suite string) string {
	// united-states/legislators formats suites as `Suite 1234` but we sometimes get back
	// just a suite number from parsing addresses
	if SuiteNumbersRegex.Match([]byte(suite)) {
		return fmt.Sprintf("Suite %s", suite)
	}

	return suite
}

func formatState(state string) string {
	return strings.ToUpper(strings.ReplaceAll(state, `.`, ``))
}

func formatPhone(phone string) string {
	// remove all non-digit characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	// special case the +1 form
	if len(digits) == 11 {
		return fmt.Sprintf("%s-%s-%s", digits[1:4], digits[4:7], digits[7:])
	}

	// if we don't have exactly 10 digits, return the original string
	if len(digits) != 10 {
		return phone
	}

	// format the phone number as xxx-xxx-xxxx
	return fmt.Sprintf("%s-%s-%s", digits[:3], digits[3:6], digits[6:])
}
