package main

import (
	"testing"
)

func TestPreserveOfficeFields_MatchingAddress(t *testing.T) {
	existing := []YAMLOffice{
		{
			ID:        "A000001-washington",
			Address:   "123 Main St.",
			Suite:     "Suite 100",
			City:      "Washington",
			State:     "DC",
			Zip:       "20001",
			Latitude:  38.9072,
			Longitude: -77.0369,
			Hours:     "M-F 9-5",
			Fax:       "202-555-1234",
			Phone:     "202-555-0000",
		},
	}

	updated := []YAMLOffice{
		{
			ID:      "A000001-washington",
			Address: "123 Main Street", // Different format, same address
			Suite:   "Suite 100",
			City:    "Washington",
			State:   "DC",
			Zip:     "20001",
			Phone:   "202-555-0000",
			// No lat/lng, hours, fax
		},
	}

	preserveOfficeFields(existing, updated)

	// Should preserve original address formatting
	if updated[0].Address != "123 Main St." {
		t.Errorf("Address not preserved: got %q, want %q", updated[0].Address, "123 Main St.")
	}

	// Should preserve lat/lng
	if updated[0].Latitude != 38.9072 {
		t.Errorf("Latitude not preserved: got %v, want %v", updated[0].Latitude, 38.9072)
	}
	if updated[0].Longitude != -77.0369 {
		t.Errorf("Longitude not preserved: got %v, want %v", updated[0].Longitude, -77.0369)
	}

	// Should preserve hours and fax
	if updated[0].Hours != "M-F 9-5" {
		t.Errorf("Hours not preserved: got %q, want %q", updated[0].Hours, "M-F 9-5")
	}
	if updated[0].Fax != "202-555-1234" {
		t.Errorf("Fax not preserved: got %q, want %q", updated[0].Fax, "202-555-1234")
	}

	// Should preserve original ID
	if updated[0].ID != "A000001-washington" {
		t.Errorf("ID not preserved: got %q, want %q", updated[0].ID, "A000001-washington")
	}
}

func TestPreserveOfficeFields_NewOffice(t *testing.T) {
	existing := []YAMLOffice{
		{
			ID:        "A000001-washington",
			Address:   "123 Main St.",
			City:      "Washington",
			State:     "DC",
			Zip:       "20001",
			Latitude:  38.9072,
			Longitude: -77.0369,
		},
	}

	updated := []YAMLOffice{
		{
			ID:      "A000001-new_york",
			Address: "456 Broadway",
			City:    "New York",
			State:   "NY",
			Zip:     "10001",
			Phone:   "212-555-0000",
		},
	}

	preserveOfficeFields(existing, updated)

	// Should NOT copy lat/lng from non-matching office
	if updated[0].Latitude != 0 {
		t.Errorf("Latitude should not be copied for new office: got %v", updated[0].Latitude)
	}
	if updated[0].Longitude != 0 {
		t.Errorf("Longitude should not be copied for new office: got %v", updated[0].Longitude)
	}

	// Should keep the new address
	if updated[0].Address != "456 Broadway" {
		t.Errorf("New office address changed: got %q, want %q", updated[0].Address, "456 Broadway")
	}
}

func TestPreserveOfficeFields_PartialMatch(t *testing.T) {
	// One matching office, one new office
	existing := []YAMLOffice{
		{
			ID:        "A000001-washington",
			Address:   "123 Main St.",
			City:      "Washington",
			State:     "DC",
			Zip:       "20001",
			Latitude:  38.9072,
			Longitude: -77.0369,
		},
	}

	updated := []YAMLOffice{
		{
			ID:      "A000001-washington",
			Address: "123 Main Street", // Matches existing
			City:    "Washington",
			State:   "DC",
			Zip:     "20001",
			Phone:   "202-555-0000",
		},
		{
			ID:      "A000001-new_york",
			Address: "456 Broadway", // New office
			City:    "New York",
			State:   "NY",
			Zip:     "10001",
			Phone:   "212-555-0000",
		},
	}

	preserveOfficeFields(existing, updated)

	// First office should have preserved fields
	if updated[0].Latitude != 38.9072 {
		t.Errorf("First office latitude not preserved: got %v", updated[0].Latitude)
	}
	if updated[0].Address != "123 Main St." {
		t.Errorf("First office address not preserved: got %q", updated[0].Address)
	}

	// Second office should not have lat/lng
	if updated[1].Latitude != 0 {
		t.Errorf("Second office should not have latitude: got %v", updated[1].Latitude)
	}
	if updated[1].Address != "456 Broadway" {
		t.Errorf("Second office address changed: got %q", updated[1].Address)
	}
}

func TestPreserveOfficeFields_SuiteVariations(t *testing.T) {
	existing := []YAMLOffice{
		{
			ID:        "A000001-washington",
			Address:   "123 Main St.",
			Suite:     "Suite 100",
			City:      "Washington",
			State:     "DC",
			Zip:       "20001",
			Latitude:  38.9072,
			Longitude: -77.0369,
		},
	}

	updated := []YAMLOffice{
		{
			ID:      "A000001-washington",
			Address: "123 Main St.",
			Suite:   "100", // Different suite format
			City:    "Washington",
			State:   "DC",
			Zip:     "20001",
			Phone:   "202-555-0000",
		},
	}

	preserveOfficeFields(existing, updated)

	// Should preserve original suite formatting
	if updated[0].Suite != "Suite 100" {
		t.Errorf("Suite not preserved: got %q, want %q", updated[0].Suite, "Suite 100")
	}

	// Should preserve lat/lng
	if updated[0].Latitude != 38.9072 {
		t.Errorf("Latitude not preserved: got %v", updated[0].Latitude)
	}
}

func TestPreserveOfficeFields_DoesNotOverwriteNewData(t *testing.T) {
	existing := []YAMLOffice{
		{
			ID:        "A000001-washington",
			Address:   "123 Main St.",
			City:      "Washington",
			State:     "DC",
			Zip:       "20001",
			Latitude:  38.9072,
			Longitude: -77.0369,
			Phone:     "202-555-0000",
			Fax:       "202-555-1111",
		},
	}

	updated := []YAMLOffice{
		{
			ID:      "A000001-washington",
			Address: "123 Main Street",
			City:    "Washington",
			State:   "DC",
			Zip:     "20001",
			Phone:   "202-555-9999", // New phone number
			Fax:     "202-555-8888", // New fax number
		},
	}

	preserveOfficeFields(existing, updated)

	// Should NOT overwrite phone/fax since updated has values
	if updated[0].Phone != "202-555-9999" {
		t.Errorf("Phone was overwritten: got %q, want %q", updated[0].Phone, "202-555-9999")
	}
	if updated[0].Fax != "202-555-8888" {
		t.Errorf("Fax was overwritten: got %q, want %q", updated[0].Fax, "202-555-8888")
	}
}

func TestPreserveOfficeFields_EmptyExisting(t *testing.T) {
	existing := []YAMLOffice{}

	updated := []YAMLOffice{
		{
			ID:      "A000001-washington",
			Address: "123 Main St.",
			City:    "Washington",
			State:   "DC",
			Zip:     "20001",
			Phone:   "202-555-0000",
		},
	}

	// Should not panic
	preserveOfficeFields(existing, updated)

	// Data should be unchanged
	if updated[0].Address != "123 Main St." {
		t.Errorf("Address changed unexpectedly: got %q", updated[0].Address)
	}
	if updated[0].Latitude != 0 {
		t.Errorf("Latitude should be 0: got %v", updated[0].Latitude)
	}
}

func TestFindEntryBounds(t *testing.T) {
	data := []byte(`- id:
    bioguide: A000001
    govtrack: 100001
  offices:
  - id: A000001-dc
    address: 123 Main St.
    city: Washington
    state: DC
    zip: '20001'
- id:
    bioguide: B000002
    govtrack: 100002
  offices:
  - id: B000002-boston
    address: 456 Oak Ave.
    city: Boston
    state: MA
    zip: '02101'
`)

	// Find first entry
	start, end, found := findEntryBounds(data, "A000001")
	if !found {
		t.Fatal("Should find A000001")
	}
	if start != 0 {
		t.Errorf("A000001 start: got %d, want 0", start)
	}

	// Find second entry
	start2, end2, found2 := findEntryBounds(data, "B000002")
	if !found2 {
		t.Fatal("Should find B000002")
	}
	if start2 != end {
		t.Errorf("B000002 should start where A000001 ends: got %d, want %d", start2, end)
	}
	if end2 != len(data) {
		t.Errorf("B000002 end should be EOF: got %d, want %d", end2, len(data))
	}

	// Not found
	_, _, found3 := findEntryBounds(data, "Z999999")
	if found3 {
		t.Error("Should not find Z999999")
	}
}

func TestFindInsertPosition(t *testing.T) {
	data := []byte(`- id:
    bioguide: A000001
    govtrack: 100001
  offices:
  - id: test
- id:
    bioguide: C000003
    govtrack: 100003
  offices:
  - id: test
`)

	// Insert between A and C
	pos := findInsertPosition(data, "B000002")
	_, endA, _ := findEntryBounds(data, "A000001")
	if pos != endA {
		t.Errorf("B000002 should insert after A000001: got %d, want %d", pos, endA)
	}

	// Insert at end
	pos2 := findInsertPosition(data, "Z999999")
	if pos2 != len(data) {
		t.Errorf("Z999999 should insert at end: got %d, want %d", pos2, len(data))
	}

	// Insert at start
	pos3 := findInsertPosition(data, "A000000")
	if pos3 != 0 {
		t.Errorf("A000000 should insert at start: got %d, want 0", pos3)
	}
}

func TestFormatEntry(t *testing.T) {
	leg := YAMLLegislatorOffices{
		ID: struct {
			Bioguide string `yaml:"bioguide"`
			Govtrack int    `yaml:"govtrack"`
			Thomas   string `yaml:"thomas,omitempty"`
		}{
			Bioguide: "A000001",
			Govtrack: 100001,
			Thomas:   "01234",
		},
		Offices: []YAMLOffice{
			{
				ID:      "A000001-dc",
				Address: "123 Main St.",
				Suite:   "100",
				City:    "Washington",
				State:   "DC",
				Zip:     "20001",
				Phone:   "202-555-0000",
			},
		},
	}

	result := string(formatEntry(leg))

	// Check structure
	if !stringContains(result, "- id:\n") {
		t.Error("Should start with '- id:'")
	}
	if !stringContains(result, "    bioguide: A000001\n") {
		t.Error("Should have bioguide")
	}
	if !stringContains(result, "    thomas: '01234'\n") {
		t.Error("Thomas should be quoted")
	}
	if !stringContains(result, "    suite: '100'\n") {
		t.Error("Numeric suite should be quoted")
	}
	if !stringContains(result, "    zip: '20001'\n") {
		t.Error("Zip should be quoted")
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
