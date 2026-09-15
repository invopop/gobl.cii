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
	const knownUNECECode = "Known UNECE code"
	tests := []struct {
		name     string
		input    string
		expected org.Unit
	}{
		{knownUNECECode, "HUR", org.Unit("h")},
		{knownUNECECode, "SEC", org.Unit("s")},
		{knownUNECECode, "MTR", org.Unit("m")},
		{knownUNECECode, "GRM", org.Unit("g")},
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

func TestCleanString(t *testing.T) {
	t.Run("leaves clean text untouched", func(t *testing.T) {
		in := `<a>Première vérification</a>`
		assert.Equal(t, in, cleanString(in))
	})

	t.Run("drops the replacement character", func(t *testing.T) {
		assert.Equal(t, "<a>Premire</a>", cleanString("<a>Premi�re</a>"))
	})

	t.Run("drops invalid UTF-8", func(t *testing.T) {
		assert.Equal(t, "<a>Premire</a>", cleanString("<a>Premi\xe9re</a>"))
	})

	t.Run("handles both at once", func(t *testing.T) {
		assert.Equal(t, "<a>n et Premire</a>", cleanString("<a>n\xe9 et Premi�re</a>"))
	})

	t.Run("drops replacement character references", func(t *testing.T) {
		// Plain ASCII in the document; only the XML decoder turns these into
		// U+FFFD, so a byte-level clean alone would miss them.
		for _, ref := range []string{"&#xFFFD;", "&#xfffd;", "&#XFFFD;", "&#x0FFFD;", "&#65533;", "&#065533;"} {
			assert.Equal(t, "<a>bad  char</a>", cleanString("<a>bad "+ref+" char</a>"), ref)
		}
	})

	t.Run("keeps an escaped reference, which is literal text", func(t *testing.T) {
		in := "<a>bad &amp;#xFFFD; char</a>"
		assert.Equal(t, in, cleanString(in))
	})

	t.Run("is idempotent", func(t *testing.T) {
		in := "<a>Premi�re</a>"
		assert.Equal(t, cleanString(in), cleanString(cleanString(in)))
	})

	t.Run("preserves valid multi-byte text", func(t *testing.T) {
		in := "<a>1000 m² – 20 €</a>"
		assert.Equal(t, in, cleanString(in))
	})
}
