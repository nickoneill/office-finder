package main

import (
	"testing"
)

func TestOfficeEquals(t *testing.T) {
	tests := []struct {
		name      string
		office    YAMLOffice
		genOffice OfficeInfo
		want      bool
	}{
		{
			name: "Exact match",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "Suite 100",
			},
			genOffice: OfficeInfo{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "Suite 100",
			},
			want: true,
		},
		{
			name: "Suite formatting",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "100",
			},
			genOffice: OfficeInfo{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "Suite 100",
			},
			want: true,
		},
		{
			name: "Suite formatting 2",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "100.3",
			},
			genOffice: OfficeInfo{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "STE 100.3",
			},
			want: true,
		},
		{
			name: "Suite formatting 3",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "Suite B",
			},
			genOffice: OfficeInfo{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "B",
			},
			want: true,
		},
		{
			name: "Different address",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "Suite 100",
			},
			genOffice: OfficeInfo{
				Address: "456 Oak Ave",
				City:    "New York",
				Suite:   "Suite 100",
			},
			want: false,
		},
		{
			name: "Different city",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "Suite 100",
			},
			genOffice: OfficeInfo{
				Address: "123 Main St",
				City:    "Los Angeles",
				Suite:   "Suite 100",
			},
			want: false,
		},
		{
			name: "Street types",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "Los Angeles",
				Suite:   "Suite 100",
			},
			genOffice: OfficeInfo{
				Address: "123 Main Street",
				City:    "Los Angeles",
				Suite:   "Suite 100",
			},
			want: true,
		},
		{
			name: "Street cardinality",
			office: YAMLOffice{
				Address: "123 Main St E",
				City:    "Los Angeles",
				Suite:   "Suite 100",
			},
			genOffice: OfficeInfo{
				Address: "123 Main St East",
				City:    "Los Angeles",
				Suite:   "Suite 100",
			},
			want: true,
		},
		{
			name: "Street cardinality + abbr",
			office: YAMLOffice{
				Address: "2000 South Stemmons Freeway",
				City:    "Los Angeles",
				Suite:   "Suite 100",
			},
			genOffice: OfficeInfo{
				Address: "2000 S. Stemmons Fwy.",
				City:    "Los Angeles",
				Suite:   "Suite 100",
			},
			want: true,
		},
		{
			name: "Different suite",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "new_york",
				Suite:   "Suite 100",
			},
			genOffice: OfficeInfo{
				Address: "123 Main St",
				City:    "New York",
				Suite:   "Suite 200",
			},
			want: false,
		},
		{
			name: "Empty suite",
			office: YAMLOffice{
				Address: "123 Main St",
				City:    "chicago",
			},
			genOffice: OfficeInfo{
				Address: "123 Main St",
				City:    "Chicago",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := officeEquals(tt.office, tt.genOffice); got != tt.want {
				t.Errorf("officeEquals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatPhone(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"(123) 456-7890", "123-456-7890"},
		{"123-456-7890", "123-456-7890"},
		{"1234567890", "123-456-7890"},
		{"123-456-789", "123-456-789"},   // Less than 10 digits, should remain unchanged
		{"+11234567891", "123-456-7891"}, // More than 10 digits, should remain unchanged
	}

	for _, tc := range testCases {
		result := formatPhone(tc.input)
		if result != tc.expected {
			t.Errorf("formatPhone(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
