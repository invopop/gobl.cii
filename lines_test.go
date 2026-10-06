package cii_test

import (
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
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
func TestLineSellerRoundTrip(t *testing.T) {
	env := loadEnvelope(t, "en16931/invoice-de-de.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	inv.Lines[0].Seller = &org.Party{
		Identities: []*org.Identity{
			{
				Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0088"}),
				Code: "1234567890128",
			},
		},
	}

	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	seller := doc.Transaction.Lines[0].Agreement.ItemSellerParty
	require.NotNil(t, seller)
	require.Len(t, seller.GlobalID, 1)
	assert.Equal(t, "0088", seller.GlobalID[0].SchemeID)
	assert.Equal(t, "1234567890128", seller.GlobalID[0].Value)

	data, err := doc.Bytes()
	require.NoError(t, err)

	parsed, err := cii.Parse(data)
	require.NoError(t, err)
	parsedInv, ok := parsed.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.NotNil(t, parsedInv.Lines[0].Seller)
	require.Len(t, parsedInv.Lines[0].Seller.Identities, 1)
	assert.Equal(t, cbc.Code("1234567890128"), parsedInv.Lines[0].Seller.Identities[0].Code)
	assert.Equal(t, cbc.Code("0088"), parsedInv.Lines[0].Seller.Identities[0].Ext.Get(iso.ExtKeySchemeID))
}

const (
	lineStatusGroup       = "GROUP"
	lineStatusDetail      = "DETAIL"
	lineStatusInformation = "INFORMATION"
)

const fixtureFacturXDE = "facturx/invoice-de-de.json"

// breakdownEnvelope loads a fixture and gives its first line a breakdown: two
// priced sub-lines, one of them discounted, and one without a price.
func breakdownEnvelope(t *testing.T, fixture string) *gobl.Envelope {
	t.Helper()
	env := loadEnvelope(t, fixture)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	design := num.MakeAmount(3000, 2)
	build := num.MakeAmount(6000, 2)
	inv.Lines[0].Discounts, inv.Lines[0].Charges = nil, nil
	inv.Lines[0].Breakdown = []*bill.SubLine{
		{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Design", Price: &design}},
		{
			Quantity:  num.MakeAmount(2, 0),
			Item:      &org.Item{Name: "Build", Price: &build},
			Discounts: []*bill.LineDiscount{{Amount: num.MakeAmount(500, 2), Reason: "Promotion"}},
		},
		{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Sample, no charge"}},
	}
	require.NoError(t, env.Calculate())
	return env
}

func TestSubLinesConvert(t *testing.T) {
	t.Run("extended profile writes sub-invoice lines", func(t *testing.T) {
		doc, err := cii.ConvertInvoice(breakdownEnvelope(t, fixtureFacturXDE), cii.WithContext(cii.ContextFacturXExtendedV1))
		require.NoError(t, err)
		lines := doc.Transaction.Lines
		require.Len(t, lines, 4)

		// The parent quantity is 20, and each sub-line counts per unit of it.
		group := lines[0]
		assert.Equal(t, "1", group.LineDoc.ID)
		assert.Equal(t, lineStatusGroup, group.LineDoc.LineStatusReasonCode)
		assert.Empty(t, group.LineDoc.ParentLineID)
		assert.Empty(t, group.TradeSettlement.ApplicableTradeTax)
		assert.Equal(t, "145.00", group.Agreement.NetPrice.Amount)
		assert.Equal(t, "20", group.Quantity.Quantity.Amount)
		assert.Equal(t, "2900.00", group.TradeSettlement.Sum.Amount)

		design := lines[1]
		assert.Equal(t, "1.1", design.LineDoc.ID)
		assert.Equal(t, "1", design.LineDoc.ParentLineID)
		assert.Equal(t, lineStatusDetail, design.LineDoc.LineStatusReasonCode)
		assert.Equal(t, "Design", design.Product.Name)
		assert.Equal(t, "20", design.Quantity.Quantity.Amount)
		assert.Equal(t, "30.00", design.Agreement.NetPrice.Amount)
		assert.Equal(t, "600.00", design.TradeSettlement.Sum.Amount)
		require.Len(t, design.TradeSettlement.ApplicableTradeTax, 1)
		assert.Equal(t, "S", design.TradeSettlement.ApplicableTradeTax[0].CategoryCode)

		build := lines[2]
		assert.Equal(t, "1.2", build.LineDoc.ID)
		assert.Equal(t, "40", build.Quantity.Quantity.Amount)
		require.Len(t, build.TradeSettlement.AllowanceCharge, 1)
		assert.Equal(t, "100.00", build.TradeSettlement.AllowanceCharge[0].Amount)
		assert.Equal(t, "2300.00", build.TradeSettlement.Sum.Amount)

		info := lines[3]
		assert.Equal(t, "1.3", info.LineDoc.ID)
		assert.Equal(t, lineStatusInformation, info.LineDoc.LineStatusReasonCode)
		assert.Equal(t, rateZero, info.Agreement.NetPrice.Amount)
		assert.Equal(t, rateZero, info.TradeSettlement.Sum.Amount)
	})

	t.Run("other profiles write the line alone", func(t *testing.T) {
		doc, err := cii.ConvertInvoice(breakdownEnvelope(t, "en16931/invoice-de-de.json"))
		require.NoError(t, err)
		require.Len(t, doc.Transaction.Lines, 1)
		assert.Empty(t, doc.Transaction.Lines[0].LineDoc.LineStatusReasonCode)
		assert.Equal(t, "2900.00", doc.Transaction.Lines[0].TradeSettlement.Sum.Amount)
	})

	t.Run("a line with its own discount is written alone", func(t *testing.T) {
		env := breakdownEnvelope(t, fixtureFacturXDE)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Lines[0].Discounts = []*bill.LineDiscount{{Amount: num.MakeAmount(1000, 2), Reason: "Loyalty"}}
		require.NoError(t, env.Calculate())

		doc, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextFacturXExtendedV1))
		require.NoError(t, err)
		require.Len(t, doc.Transaction.Lines, 1)
	})

	t.Run("unpriced sub-lines describe a line that keeps its price", func(t *testing.T) {
		env := loadEnvelope(t, fixtureFacturXDE)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Lines[0].Breakdown = []*bill.SubLine{
			{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Helmet"}},
		}
		require.NoError(t, env.Calculate())

		doc, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextFacturXExtendedV1))
		require.NoError(t, err)
		lines := doc.Transaction.Lines
		require.Len(t, lines, 2)
		assert.Equal(t, lineStatusDetail, lines[0].LineDoc.LineStatusReasonCode)
		assert.NotEmpty(t, lines[0].TradeSettlement.ApplicableTradeTax)
		assert.Equal(t, "1800.00", lines[0].TradeSettlement.Sum.Amount)
		assert.Equal(t, lineStatusInformation, lines[1].LineDoc.LineStatusReasonCode)
	})
}

// TestSubLinesRoundTrip writes a breakdown out as sub-invoice lines and reads
// it back: the breakdown and every total must survive.
func TestSubLinesRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		env     func(t *testing.T) *gobl.Envelope
		context cii.Context
	}{
		{"invoice-hierarchy", func(t *testing.T) *gobl.Envelope {
			return loadEnvelope(t, "peppol-france-facturx/invoice-hierarchy.json")
		}, cii.ContextPeppolFranceFacturXV1},
		{"invoice-sub-lines", func(t *testing.T) *gobl.Envelope {
			return loadEnvelope(t, "peppol-france-extended/invoice-sub-lines.json")
		}, cii.ContextPeppolFranceExtendedV1},
		{"facturx-extended", func(t *testing.T) *gobl.Envelope {
			return breakdownEnvelope(t, fixtureFacturXDE)
		}, cii.ContextFacturXExtendedV1},
		{"zugferd-extended", func(t *testing.T) *gobl.Envelope {
			return breakdownEnvelope(t, "zugferd/standard-invoice.json")
		}, cii.ContextZUGFeRDExtendedV2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.env(t)
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)

			doc, err := cii.ConvertInvoice(env, cii.WithContext(tt.context))
			require.NoError(t, err)
			data, err := doc.Bytes()
			require.NoError(t, err)

			parsed, err := cii.Parse(data)
			require.NoError(t, err)
			out, ok := parsed.Extract().(*bill.Invoice)
			require.True(t, ok)

			assert.False(t, out.HasTags(tax.TagBypass))
			require.Len(t, out.Lines, len(inv.Lines))
			for i, l := range inv.Lines {
				got := out.Lines[i]
				assert.Equal(t, l.Quantity.String(), got.Quantity.String(), "line %d quantity", i+1)
				assert.Equal(t, l.Total.String(), got.Total.String(), "line %d total", i+1)
				require.Len(t, got.Breakdown, len(l.Breakdown), "line %d breakdown", i+1)
				for j, sl := range l.Breakdown {
					gs := got.Breakdown[j]
					assert.Equal(t, sl.Item.Name, gs.Item.Name)
					assert.True(t, sl.Quantity.Equals(gs.Quantity), "sub-line %d.%d quantity", i+1, j+1)
					if sl.Item.Price == nil {
						assert.Nil(t, gs.Item.Price)
						assert.Nil(t, gs.Total)
						continue
					}
					require.NotNil(t, gs.Item.Price)
					assert.True(t, sl.Item.Price.Equals(*gs.Item.Price), "sub-line %d.%d price", i+1, j+1)
					assert.Equal(t, sl.Total.String(), gs.Total.String(), "sub-line %d.%d total", i+1, j+1)
					assert.Len(t, gs.Discounts, len(sl.Discounts))
				}
			}
			assert.Equal(t, inv.Totals.Sum.String(), out.Totals.Sum.String())
			assert.Equal(t, inv.Totals.Tax.String(), out.Totals.Tax.String())
			assert.Equal(t, inv.Totals.Payable.String(), out.Totals.Payable.String())
		})
	}
}
