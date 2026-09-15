package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLines(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "1", doc.Transaction.Lines[0].LineDoc.ID)
		assert.Equal(t, "Development services", doc.Transaction.Lines[0].Product.Name)
		assert.Equal(t, "90.00", doc.Transaction.Lines[0].Agreement.NetPrice.Amount)
		assert.Equal(t, "20", doc.Transaction.Lines[0].Quantity.Quantity.Amount)
		assert.Equal(t, "HUR", doc.Transaction.Lines[0].Quantity.Quantity.UnitCode)
		assert.Equal(t, "VAT", doc.Transaction.Lines[0].TradeSettlement.ApplicableTradeTax[0].TypeCode)
		assert.Equal(t, "19", doc.Transaction.Lines[0].TradeSettlement.ApplicableTradeTax[0].RateApplicablePercent)
		assert.Equal(t, "1800.00", doc.Transaction.Lines[0].TradeSettlement.Sum.Amount)
		assert.Equal(t, "123456789", doc.Transaction.Lines[0].Product.GlobalID.Value)
		assert.Equal(t, "0088", doc.Transaction.Lines[0].Product.GlobalID.SchemeID)
		assert.Equal(t, "20240912", doc.Transaction.Lines[0].TradeSettlement.Period.Start.DateFormat.Value)
		assert.Equal(t, "20241012", doc.Transaction.Lines[0].TradeSettlement.Period.End.DateFormat.Value)
	})

}

func TestLineNoteSubjectCodeRoundTrip(t *testing.T) {
	env := loadEnvelope(t, "facturx/invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	inv.Lines[0].Notes = []*org.Note{
		{
			Text: "Handle with care",
			Ext:  tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTextSubject: "AAI"}),
		},
	}

	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	require.NotEmpty(t, doc.Transaction.Lines[0].LineDoc.Note)
	assert.Equal(t, "Handle with care", doc.Transaction.Lines[0].LineDoc.Note[0].Content)
	assert.Equal(t, "AAI", doc.Transaction.Lines[0].LineDoc.Note[0].SubjectCode)

	data, err := doc.Bytes()
	require.NoError(t, err)

	outEnv, err := cii.Parse(data)
	require.NoError(t, err)
	outInv, ok := outEnv.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.NotEmpty(t, outInv.Lines[0].Notes)
	n := outInv.Lines[0].Notes[0]
	assert.Equal(t, "Handle with care", n.Text)
	assert.Equal(t, cbc.Code("AAI"), n.Ext.Get(untdid.ExtKeyTextSubject))
}

func TestItemAttributeRoundTrip(t *testing.T) {
	env := loadEnvelope(t, "facturx/invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	weight := num.MakeAmount(25, 1) // 2.5
	inv.Lines[0].Item.Attributes = []*org.Attribute{
		{Label: attrLabelColor, Text: attrValueBlack},
		{Label: attrLabelWeight, Amount: &weight, Unit: org.UnitKilogram},
	}
	require.NoError(t, env.Calculate())

	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	chars := doc.Transaction.Lines[0].Product.Characteristics
	require.Len(t, chars, 2)

	assert.Equal(t, attrLabelColor, chars[0].Description)
	assert.Equal(t, attrValueBlack, chars[0].Value)

	assert.Equal(t, attrLabelWeight, chars[1].Description)
	// CII has nowhere to put the unit other than the value itself.
	assert.Equal(t, "2.5 kg", chars[1].Value)
	assert.Nil(t, chars[1].ValueMeasure)

	data, err := doc.Bytes()
	require.NoError(t, err)

	outEnv, err := cii.Parse(data)
	require.NoError(t, err)
	outInv, ok := outEnv.Extract().(*bill.Invoice)
	require.True(t, ok)

	attrs := outInv.Lines[0].Item.Attributes
	require.Len(t, attrs, 2)
	assert.Equal(t, attrLabelColor, attrs[0].Label)
	assert.Equal(t, attrValueBlack, attrs[0].Text)
	// The amount comes back as the text CII carried it in.
	assert.Equal(t, attrLabelWeight, attrs[1].Label)
	assert.Equal(t, "2.5 kg", attrs[1].Text)
	assert.Nil(t, attrs[1].Amount)
}

// TestItemAttributeMeasureParse covers the measure a sender using the CII
// extended profile may provide, which GOBL reads even though it never
// writes one.
func TestItemAttributeMeasureParse(t *testing.T) {
	env := loadEnvelope(t, "facturx/invoice-minimal.json")
	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	doc.Transaction.Lines[0].Product.Characteristics = []*cii.Characteristic{
		{
			Description:  attrLabelWeight,
			ValueMeasure: &cii.Quantity{Amount: "2.5", UnitCode: "KGM"},
			Value:        "2.5 kg",
		},
	}
	data, err := doc.Bytes()
	require.NoError(t, err)

	outEnv, err := cii.Parse(data)
	require.NoError(t, err)
	outInv, ok := outEnv.Extract().(*bill.Invoice)
	require.True(t, ok)

	attrs := outInv.Lines[0].Item.Attributes
	require.Len(t, attrs, 1)
	assert.Equal(t, attrLabelWeight, attrs[0].Label)
	require.NotNil(t, attrs[0].Amount)
	assert.Equal(t, "2.5", attrs[0].Amount.String())
	assert.Equal(t, org.UnitKilogram, attrs[0].Unit)
	assert.Equal(t, cbc.Code("KGM"), attrs[0].Ext.Get(untdid.ExtKeyUnit), "the document stated the code")
}

// TestItemAttributeUnmappedUnit covers a UN/ECE unit code that GOBL has no key
// for, which is preserved in the attribute's extensions.
func TestItemAttributeUnmappedUnit(t *testing.T) {
	env := loadEnvelope(t, "facturx/invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	length := num.MakeAmount(25, 1) // 2.5
	inv.Lines[0].Item.Attributes = []*org.Attribute{
		{
			Label:  "Length",
			Amount: &length,
			Ext:    tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyUnit: "X4G"}),
		},
	}
	require.NoError(t, env.Calculate())

	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	chars := doc.Transaction.Lines[0].Product.Characteristics
	require.Len(t, chars, 1)
	// With no GOBL key the raw code stands in as the presentation label.
	assert.Equal(t, "2.5 X4G", chars[0].Value)
}

// TestItemAttributeUnitWithoutUNTDID covers a GOBL unit that has no UN/ECE
// equivalent, which still names itself in the value.
func TestItemAttributeUnitWithoutUNTDID(t *testing.T) {
	env := loadEnvelope(t, "facturx/invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	size := num.MakeAmount(25, 1) // 2.5
	inv.Lines[0].Item.Attributes = []*org.Attribute{
		{Label: "Serving", Amount: &size, Unit: org.UnitPortion},
	}
	require.NoError(t, env.Calculate())

	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	chars := doc.Transaction.Lines[0].Product.Characteristics
	require.Len(t, chars, 1)
	assert.Equal(t, "2.5 portion", chars[0].Value)
}

// TestItemAttributeKeyName covers an attribute identified by its key, which
// names the characteristic when no label is given.
func TestItemAttributeKeyName(t *testing.T) {
	env := loadEnvelope(t, "facturx/invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	inv.Lines[0].Item.Attributes = []*org.Attribute{
		{Key: org.AttributeKeyColor, Text: "Black"},
	}
	require.NoError(t, env.Calculate())

	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	chars := doc.Transaction.Lines[0].Product.Characteristics
	require.Len(t, chars, 1)
	assert.Equal(t, "color", chars[0].Description)
	assert.Equal(t, attrValueBlack, chars[0].Value)
}
