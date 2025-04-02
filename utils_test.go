package cii

import (
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
)

// Define tests for the ParseDate function
func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"Valid date", "20230515", "2023-05-15", false},
		{"Invalid date", "20231345", "", true},
		{"Empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDate(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

// Define tests for the TypeCodeParse function
func TestTypeCodeParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Standard invoice", "380", "standard"},
		{"Credit note", "381", "credit-note"},
		{"Corrective invoice", "384", "corrective"},
		{"Proforma invoice", "325", "proforma"},
		{"Debit note", "383", "debit-note"},
		{"Unknown type code", "999", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeCodeParse(tt.input)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// Define tests for the UnitFromUNECE function
func TestUnitFromUNECE(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected org.Unit
	}{
		{"Known UNECE code", "HUR", org.Unit("h")},
		{"Known UNECE code", "SEC", org.Unit("s")},
		{"Known UNECE code", "MTR", org.Unit("m")},
		{"Known UNECE code", "GRM", org.Unit("g")},
		{"Unknown UNECE code", "XYZ", org.Unit("XYZ")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := cbc.Code(tt.input)
			result := goblUnitFromUNECE(code)
			assert.Equal(t, tt.expected, result)
		})
	}
}
