package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

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

				for _, generatedOffice := range generatedOffices.Offices {
					// skip any main offices in dc
					if strings.ToLower(generatedOffice.City) == "washington" || strings.ToLower(generatedOffice.State) == "d.c." {
						continue
					}
					officeKey := fmt.Sprintf("%s-%s", generatedOffices.Bioguide, normalizeCity(generatedOffice.City))

					exists := false
					for _, office := range legislator.Offices {
						if office.ID == officeKey {
							exists = true
						}
					}
					if !exists {
						statsNewOffices++
						log.Printf("new office: %s", officeKey)
					}
				}

				for _, existingOffice := range legislator.Offices {
					exists := false
					for _, generatedOffice := range generatedOffices.Offices {
						officeKey := fmt.Sprintf("%s-%s", generatedOffices.Bioguide, normalizeCity(generatedOffice.City))
						if existingOffice.ID == officeKey {
							exists = true
						}
					}
					if !exists {
						statsRemovedOffices++
						log.Printf("removed office: %s", existingOffice.ID)
					}
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

func normalizeCity(city string) string {
	// replace spaces or periods with underscores (yes, st__george is the right style for this key)
	return strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(city), " ", "_"), ".", "_")

}
