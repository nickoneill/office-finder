package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	Latitude  float64 `yaml:"latitude,omitempty"`
	Longitude float64 `yaml:"longitude,omitempty"`
	Phone     string  `yaml:"phone,omitempty"`
	Fax       string  `yaml:"fax,omitempty"`
	Hours     string  `yaml:"hours,omitempty"`
}

func upstreamChanges() error {
	officesData, err := os.ReadFile("offices.json")
	if err != nil {
		return fmt.Errorf("error reading offices.json: %v", err)
	}

	var officeList []OfficeList
	err = json.Unmarshal(officesData, &officeList)
	if err != nil {
		return fmt.Errorf("error parsing offices.json: %v", err)
	}

	resp, err := http.Get("https://raw.githubusercontent.com/unitedstates/congress-legislators/main/legislators-district-offices.yaml")
	if err != nil {
		return fmt.Errorf("error fetching YAML file: %v", err)
	}
	defer resp.Body.Close()

	yamlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading YAML data: %v", err)
	}

	var legislators []YAMLLegislatorOffices
	err = yaml.Unmarshal(yamlData, &legislators)
	if err != nil {
		return fmt.Errorf("error parsing YAML data: %v", err)
	}

	statsNewOffices := 0
	statsRemovedOffices := 0
	statsNewLegislators := 0

	// Create a map to keep track of processed bioguides
	processedBioguides := make(map[string]bool)

	// search through each list to match the office lists to compare
	for li, _ := range legislators {
		for _, generatedOffices := range officeList {
			if legislators[li].ID.Bioguide == generatedOffices.Bioguide {
				processedBioguides[generatedOffices.Bioguide] = true
				log.Printf("%s %s:", generatedOffices.URL, generatedOffices.Bioguide)
				// now we have the right set of offices, check which ones already exist and which ones need to be created or removed

				// loop through the existing offices
				// * check each office against the generated ones by comparing address, suite, city
				// * if no offices match, remove them
				// * if an office matches, remove from the generated list
				// * add any leftover generated offices to the list at the end
				// * * ignore leftover washington offices
				// * * ensure duplicate office keys get `-1`,`-2` etc

				genOfficesCopy := generatedOffices.Offices
				// loop both office lists in reverse so we can remove any items that have been found
				for i := len(legislators[li].Offices) - 1; i >= 0; i-- {
					isFound := false
					for j := len(genOfficesCopy) - 1; j >= 0; j-- {
						if officeEquals(legislators[li].Offices[i], genOfficesCopy[j]) {
							isFound = true
							genOfficesCopy = append(genOfficesCopy[:j], genOfficesCopy[j+1:]...)
						}
					}

					if !isFound {
						log.Printf("removing office in %s", legislators[li].Offices[i].City)
						statsRemovedOffices++
						legislators[li].Offices = append(legislators[li].Offices[:i], legislators[li].Offices[i+1:]...)
					}
				}
				for _, remainingGenOffice := range genOfficesCopy {
					// skip any main offices in dc
					if strings.ToLower(remainingGenOffice.City) == "washington" || strings.ToLower(remainingGenOffice.State) == "d.c." || strings.ToLower(remainingGenOffice.State) == "dc" {
						// special casing one office for EHN until I can think of a better way to handle this
						if generatedOffices.Bioguide == "N000147" && strings.HasPrefix(remainingGenOffice.Address, "1300 Pennsylvania") {
							// don't skip, add the office
						} else {
							continue
						}
					}

					log.Printf("adding office in %s", remainingGenOffice.City)
					statsNewOffices++
					legislators[li].Offices = append(legislators[li].Offices, officeFromGenOffice(remainingGenOffice, legislators[li].ID.Bioguide, legislators[li].Offices))
				}
			}
		}
	}

	// Process any remaining legislators and offices from officeList
	for _, generatedOffices := range officeList {
		if !processedBioguides[generatedOffices.Bioguide] {
			log.Printf("Adding new legislator: %s", generatedOffices.Bioguide)
			statsNewLegislators++
			newLegislator := YAMLLegislatorOffices{
				ID: struct {
					Bioguide string `yaml:"bioguide"`
					Govtrack int    `yaml:"govtrack"`
					Thomas   string `yaml:"thomas,omitempty"`
				}{
					Bioguide: generatedOffices.Bioguide,
					// TODO: Look up the govtrack ID
				},
				Offices: []YAMLOffice{},
			}

			for _, office := range generatedOffices.Offices {
				// Skip Washington DC offices as before
				if strings.ToLower(office.City) == "washington" || strings.ToLower(office.State) == "d.c." || strings.ToLower(office.State) == "dc" {
					continue
				}
				newLegislator.Offices = append(newLegislator.Offices, officeFromGenOffice(office, generatedOffices.Bioguide, newLegislator.Offices))
				statsNewOffices++
			}

			legislators = append(legislators, newLegislator)
		}
	}

	log.Printf("found %d new offices, removed %d old offices, added %d new legislators", statsNewOffices, statsRemovedOffices, statsNewLegislators)

	updatedYAML, err := yaml.Marshal(legislators)
	if err != nil {
		return fmt.Errorf("error marshaling updated YAML data: %v", err)
	}

	// we want single quoted strings for zips and numeric IDs so replace them all here
	singleQuotedUpdatedYAML := strings.ReplaceAll(string(updatedYAML), `"`, `'`)

	err = os.WriteFile("updated_legislators-district-offices.yaml", []byte(singleQuotedUpdatedYAML), 0644)
	if err != nil {
		return fmt.Errorf("error writing updated YAML file: %v", err)
	}

	fmt.Println("Updated YAML file has been created: updated_legislators-district-offices.yaml")
	return nil
}

func cityKey(city string) string {
	// replace spaces or periods with underscores (yes, st__george is the right style for this key)
	return strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(city), " ", "_"), ".", "_")
}

func officeEquals(office YAMLOffice, genOffice OfficeInfo) bool {
	// why is this not just existing YAMLOffice deepequals officeFromGenOffice?
	// * don't want to compare stuff like office id
	// * want to be fuzzy for stuff like St. / Street
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

// note that we need the existing offices to return cases where the offices are in the same city and
// have keys like `philadelphia-1`, `philadelphia-2`
func officeFromGenOffice(genOffice OfficeInfo, bioguide string, existingOffices []YAMLOffice) YAMLOffice {
	return YAMLOffice{
		ID:       nextOfficeKey(bioguide, genOffice.City, existingOffices),
		Address:  genOffice.Address,
		City:     genOffice.City,
		Suite:    formatSuite(genOffice.Suite),
		Building: genOffice.Building,
		Zip:      genOffice.Zip,
		State:    formatState(genOffice.State),
		Phone:    formatPhone(genOffice.Phone),
		Fax:      formatPhone(genOffice.Fax),
	}
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
