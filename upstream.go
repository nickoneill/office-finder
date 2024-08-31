package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// sometimes suite numbers contain dots, or letters
var SuiteNumbersRegex = regexp.MustCompile(`^[a-z0-9\.]+$`)

type YAMLLegislatorOffices struct {
	ID struct {
		Bioguide string `yaml:"bioguide"`
		Govtrack string `yaml:"govtrack"`
		Thomas   string `yaml:"thomas"`
	} `yaml:"id"`
	Offices []YAMLOffice `yaml:"offices"`
}

type YAMLOffice struct {
	ID       string `yaml:"id"`
	Address  string `yaml:"address"`
	Building string `yaml:"building"`
	City     string `yaml:"city"`
	Fax      string `yaml:"fax"`
	Phone    string `yaml:"phone"`
	State    string `yaml:"state"`
	Suite    string `yaml:"suite"`
	Zip      string `yaml:"zip"`
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
	// search through each list to match the office lists to compare
	for _, legislator := range legislators {
		for _, generatedOffices := range officeList {
			if legislator.ID.Bioguide == generatedOffices.Bioguide {
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
				for i := len(legislator.Offices) - 1; i >= 0; i-- {
					isFound := false
					for j := len(genOfficesCopy) - 1; j >= 0; j-- {
						if officeEquals(legislator.Offices[i], genOfficesCopy[j]) {
							isFound = true
							genOfficesCopy = append(genOfficesCopy[:j], genOfficesCopy[j+1:]...)
						}
					}

					if !isFound {
						statsRemovedOffices++
						legislator.Offices = append(legislator.Offices[:i], legislator.Offices[i+1:]...)
					}
				}
				for _, remainingGenOffice := range genOfficesCopy {
					// skip any main offices in dc
					if strings.ToLower(remainingGenOffice.City) == "washington" || strings.ToLower(remainingGenOffice.State) == "d.c." {
						continue
					}

					statsNewOffices++
					legislator.Offices = append(legislator.Offices, officeFromGenOffice(remainingGenOffice, legislator.ID.Bioguide, legislator.Offices))
				}
			}
		}
	}

	log.Printf("found %d new offices, removed %d old offices", statsNewOffices, statsRemovedOffices)

	// updatedYAML, err := yaml.Marshal(legislators)
	// if err != nil {
	// 	return fmt.Errorf("error marshaling updated YAML data: %v", err)
	// }

	// err = os.WriteFile("updated_legislators-district-offices.yaml", updatedYAML, 0644)
	// if err != nil {
	// 	return fmt.Errorf("error writing updated YAML file: %v", err)
	// }

	// fmt.Println("Updated YAML file has been created: updated_legislators-district-offices.yaml")
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

	if !sameAddress {
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
		State:    genOffice.State,
		Phone:    genOffice.Phone,
		Fax:      genOffice.Fax,
	}
}

func nextOfficeKey(bioguide, city string, existingOffices []YAMLOffice) string {
	return fmt.Sprintf("%s-%s", bioguide, cityKey(city))
}

func formatSuite(suite string) string {
	// united-states/legislators formats suites as `Suite 1234` but we sometimes get back
	// just a suite number from parsing addresses
	if SuiteNumbersRegex.Match([]byte(suite)) {
		return fmt.Sprintf("Suite %s", suite)
	}

	return suite
}
