package cii

import (
	"testing"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAttrWeight   = "Weight"
	testUnitUnmapped = "XYZ"
)

func TestUntdidUnit(t *testing.T) {
	const knownUNECECode = "Known UNECE code"
	tests := []struct {
		name     string
		item     *org.Item
		expected cbc.Code
	}{
		{knownUNECECode, &org.Item{Unit: org.UnitHour}, "HUR"},
		{knownUNECECode, &org.Item{Unit: org.UnitSecond}, "SEC"},
		{knownUNECECode, &org.Item{Unit: org.UnitMetre}, "MTR"},
		{knownUNECECode, &org.Item{Unit: org.UnitGram}, "GRM"},
		{"Unit without a UNECE code", &org.Item{Unit: cbc.Key("foo")}, ""},
		{"Code GOBL has no unit for", &org.Item{
			Ext: tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyUnit: testUnitUnmapped}),
		}, testUnitUnmapped},
		{"Unit takes priority", &org.Item{
			Unit: org.UnitHour,
			Ext:  tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyUnit: testUnitUnmapped}),
		}, "HUR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, untdidUnit(tt.item.Ext, tt.item.Unit))
		})
	}
}

func TestUnitLabel(t *testing.T) {
	assert.Equal(t, "kg", unitLabel(org.UnitKilogram, "KGM"))
	assert.Equal(t, "X4G", unitLabel(cbc.KeyEmpty, "X4G"))
	assert.Empty(t, unitLabel(cbc.KeyEmpty, cbc.CodeEmpty))
}

func TestNewCharacteristics(t *testing.T) {
	amount := num.MakeAmount(25, 1) // 2.5
	date := cal.MakeDate(2026, 9, 15)
	chars := newCharacteristics([]*org.Attribute{
		{Label: testAttrWeight, Amount: &amount, Unit: org.UnitKilogram},
		{Key: org.AttributeKeyColor, Text: "Black"},
		{Type: "AAB", Code: "RAL5010"},
		{Label: "Expiry", Date: &date},
		{Label: "Empty"},
	})

	require.Len(t, chars, 4)
	assert.Equal(t, testAttrWeight, chars[0].Description)
	assert.Equal(t, "2.5 kg", chars[0].Value)
	assert.Equal(t, "color", chars[1].Description)
	assert.Equal(t, "Black", chars[1].Value)
	assert.Equal(t, "AAB", chars[2].Description)
	assert.Equal(t, "RAL5010", chars[2].Value)
	assert.Equal(t, "Expiry", chars[3].Description)
	assert.Equal(t, "2026-09-15", chars[3].Value)

	assert.Nil(t, newCharacteristics(nil))
}

func TestGoblItemAttribute(t *testing.T) {
	attr, err := goblItemAttribute(&Characteristic{Value: "Black"})
	require.NoError(t, err)
	assert.Nil(t, attr, "a characteristic without a description has no name")

	attr, err = goblItemAttribute(&Characteristic{Description: "Color"})
	require.NoError(t, err)
	assert.Nil(t, attr, "a characteristic without a value has nothing to hold")

	_, err = goblItemAttribute(&Characteristic{
		Description:  testAttrWeight,
		ValueMeasure: &Quantity{Amount: testNotANumber, UnitCode: "KGM"},
	})
	assert.Error(t, err)

	attr, err = goblItemAttribute(&Characteristic{
		Description:  testAttrWeight,
		ValueMeasure: &Quantity{Amount: "2.5"},
	})
	require.NoError(t, err)
	require.NotNil(t, attr)
	assert.Equal(t, "2.5", attr.Amount.String())
	assert.Equal(t, cbc.KeyEmpty, attr.Unit, "a measure without a unit code has no unit")
}
