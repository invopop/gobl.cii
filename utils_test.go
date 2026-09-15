package cii

import (
	"testing"

	"github.com/invopop/gobl/catalogues/untdid"
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

func TestGoblItemUnit(t *testing.T) {
	t.Run("known code", func(t *testing.T) {
		item := new(org.Item)
		goblItemUnit(item, "HUR")
		assert.Equal(t, org.UnitHour, item.Unit)
		assert.Equal(t, cbc.Code("HUR"), item.Ext.Get(untdid.ExtKeyUnit))
	})
	t.Run("unknown code is kept in the extension", func(t *testing.T) {
		item := new(org.Item)
		goblItemUnit(item, "XYZ")
		assert.Empty(t, item.Unit)
		assert.Equal(t, cbc.Code("XYZ"), item.Ext.Get(untdid.ExtKeyUnit))
	})
	t.Run("empty code is ignored", func(t *testing.T) {
		item := new(org.Item)
		goblItemUnit(item, "")
		assert.Empty(t, item.Unit)
		assert.False(t, item.Ext.Has(untdid.ExtKeyUnit))
	})
}
