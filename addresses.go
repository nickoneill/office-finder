package main

import (
	"regexp"
	"strings"
)

var streetTypeAbbreviations = map[string]string{
	"street":    "st",
	"avenue":    "ave",
	"boulevard": "blvd",
	"drive":     "dr",
	"lane":      "ln",
	"road":      "rd",
	"circle":    "cir",
	"court":     "ct",
	"place":     "pl",
	"square":    "sq",
	"terrace":   "ter",
	"way":       "way",
	"parkway":   "pkwy",
	"freeway":   "fwy",
	"highway":   "hwy",
	"plaza":     "plz",
}

var cardinalityAbbreviations = map[string]string{
	"east":      "e",
	"west":      "w",
	"north":     "n",
	"south":     "s",
	"southwest": "sw",
	"northeast": "ne",
	"southeast": "se",
	"northwest": "nw",
}

// normalize addresses for comparison, removing spaces and punctuation and always using abbreviations
// for street types or cardinality
func normalizeAddress(address string) string {
	address = strings.ToLower(address)

	reg := regexp.MustCompile(`[^\w\s]`)
	address = reg.ReplaceAllString(address, "")

	words := strings.Fields(address)
	for i, word := range words {
		if abbr, ok := streetTypeAbbreviations[word]; ok {
			words[i] = abbr
		}

		if abbr, ok := cardinalityAbbreviations[word]; ok {
			words[i] = abbr
		}
	}

	return strings.Join(words, " ")
}

// normalize suite by only returning the numbers at the end of the string
// ...and dots, some suites have dots
func normalizeSuite(suite string) string {
	reg := regexp.MustCompile(`[\.\d]+$`)

	return reg.FindString(suite)
}

func normalizeCity(city string) string {
	return strings.ToLower(city)
}
