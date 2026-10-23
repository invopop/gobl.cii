package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test formats registered on top of the base ones, with identifiers no real
// document uses.
var (
	formatTestMatch = cii.Format{
		Key:         "cii+en16931-v2017+test-match",
		Name:        i18n.NewString("CII Test Match"),
		Import:      cii.FormatEN16931.Import,
		Export:      cii.FormatEN16931.Export,
		GuidelineID: "urn:test:match",
		Addons:      cii.FormatEN16931.Addons,
		Match: func(guidelineID, businessID string) bool {
			return guidelineID == cii.GuidelineIDEN16931 && businessID == "test-match"
		},
	}
	formatTestOutput = cii.Format{
		Key:               "cii+en16931-v2017+test-output",
		Name:              i18n.NewString("CII Test Output"),
		Import:            cii.FormatEN16931.Import,
		Export:            cii.FormatEN16931.Export,
		GuidelineID:       "urn:test:output-in",
		OutputGuidelineID: "urn:test:output-out",
		Addons:            cii.FormatEN16931.Addons,
	}
	formatTestFallback = cii.Format{
		Key:         "cii+en16931-v2017+test-fallback",
		Name:        i18n.NewString("CII Test Fallback"),
		Import:      cii.FormatEN16931.Import,
		Export:      cii.FormatEN16931.Export,
		GuidelineID: "urn:test:fallback",
		Addons:      cii.FormatEN16931.Addons,
		Fallback: func(_, businessID string) bool {
			return businessID == "test-fallback"
		},
	}
	formatTestExportOnly = cii.Format{
		Key:         "cii+en16931-v2017+test-export-only",
		Name:        i18n.NewString("CII Test Export Only"),
		Export:      cii.FormatEN16931.Export,
		GuidelineID: "urn:test:export-only",
		Addons:      cii.FormatEN16931.Addons,
	}
	formatTestNamed = cii.Format{
		Key:         "cii+en16931-v2017+test-named",
		Name:        i18n.NewString("CII Test Named"),
		Import:      cii.FormatEN16931.Import,
		Export:      cii.FormatEN16931.Export,
		GuidelineID: "urn:test:named",
		Addons:      cii.FormatEN16931.Addons,
		Named:       true,
	}
)

func init() {
	cii.RegisterFormats(formatTestMatch, formatTestOutput, formatTestFallback, formatTestExportOnly, formatTestNamed)
}

func TestFindFormat(t *testing.T) {
	tests := []struct {
		name        string
		guidelineID string
		businessID  string
		want        cbc.Key
	}{
		{"EN 16931 by guideline", cii.GuidelineIDEN16931, "", cii.FormatEN16931.Key},
		{"EN 16931 with another business process", cii.GuidelineIDEN16931, "some-business-process", cii.FormatEN16931.Key},
		{"Peppol by guideline and business process", cii.FormatPeppol.GuidelineID, cii.ProfileIDPeppolBilling, cii.FormatPeppol.Key},
		{"Peppol with a different business process", cii.FormatPeppol.GuidelineID, "other", cbc.KeyEmpty},
		{"match function", cii.GuidelineIDEN16931, "test-match", formatTestMatch.Key},
		{"output guideline", "urn:test:output-out", "", formatTestOutput.Key},
		{"fallback function", "urn:test:unknown", "test-fallback", formatTestFallback.Key},
		{"export only", "urn:test:export-only", "", cbc.KeyEmpty},
		{"named", "urn:test:named", "", cbc.KeyEmpty},
		{"unknown", "urn:test:unknown", "", cbc.KeyEmpty},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cii.FindFormat(tt.guidelineID, tt.businessID)
			if tt.want == cbc.KeyEmpty {
				assert.Nil(t, f)
				return
			}
			require.NotNil(t, f)
			assert.Equal(t, tt.want, f.Key)
		})
	}
}

func TestFormatFor(t *testing.T) {
	f := cii.FormatFor("cii+peppol+bis-v3")
	require.NotNil(t, f)
	assert.Equal(t, cii.FormatPeppol.GuidelineID, f.GuidelineID)
	assert.Nil(t, cii.FormatFor("cii+unknown"))
}

func TestFormatIs(t *testing.T) {
	f := cii.FormatEN16931
	assert.True(t, f.Is(cii.FormatEN16931))
	assert.False(t, f.Is(cii.FormatPeppol))
}

func TestRegisterFormats(t *testing.T) {
	f := convert.FormatFor(formatTestOutput.Key)
	require.NotNil(t, f)
	assert.Equal(t, cbc.Key("cii"), f.Syntax)
	assert.Equal(t, formatTestOutput.Import, f.Import)
	assert.Equal(t, formatTestOutput.Export, f.Export)

	t.Run("export only", func(t *testing.T) {
		f := convert.FormatFor(formatTestExportOnly.Key)
		require.NotNil(t, f)
		assert.Empty(t, f.Import)
		assert.Equal(t, formatTestExportOnly.Export, f.Export)
	})
}
