package cii_test

import (
	"testing"

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
	env := loadEnvelope(t, "en16931/invoice-de-de.json")
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
	note := doc.Transaction.Lines[0].LineDoc.Note[0]
	assert.Equal(t, "Handle with care", note.Content)
	assert.Equal(t, "AAI", note.SubjectCode)

	data, err := doc.Bytes()
	require.NoError(t, err)

	parsed, err := cii.Parse(data)
	require.NoError(t, err)
	parsedInv, ok := parsed.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.NotEmpty(t, parsedInv.Lines[0].Notes)
	n := parsedInv.Lines[0].Notes[0]
	assert.Equal(t, "Handle with care", n.Text)
	assert.Equal(t, cbc.Code("AAI"), n.Ext.Get(untdid.ExtKeyTextSubject))
}

func TestItemAttributeRoundTrip(t *testing.T) {
	env := loadEnvelope(t, "en16931/invoice-de-de.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	weight := num.MakeAmount(25, 1) // 2.5
	inv.Lines[0].Item.Attributes = []*org.Attribute{
		{Label: "Color", Text: "Black"},
		{Label: "Weight", Amount: &weight, Unit: "kg"},
	}

	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	require.Len(t, doc.Transaction.Lines[0].Product.Characteristics, 2)

	char1 := doc.Transaction.Lines[0].Product.Characteristics[0]
	assert.Equal(t, "Color", char1.Description)
	assert.Equal(t, "Black", char1.Value)

	// CII has no element for a quantity+unit value (unlike UBL's
	// ValueQuantity), so it's rendered as plain text.
	char2 := doc.Transaction.Lines[0].Product.Characteristics[1]
	assert.Equal(t, "Weight", char2.Description)
	assert.Equal(t, "2.5 kg", char2.Value)

	data, err := doc.Bytes()
	require.NoError(t, err)

	parsed, err := cii.Parse(data)
	require.NoError(t, err)
	parsedInv, ok := parsed.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.Len(t, parsedInv.Lines[0].Item.Attributes, 2)
	assert.Equal(t, "Color", parsedInv.Lines[0].Item.Attributes[0].Label)
	assert.Equal(t, "Black", parsedInv.Lines[0].Item.Attributes[0].Text)
	assert.Equal(t, "Weight", parsedInv.Lines[0].Item.Attributes[1].Label)
	assert.Equal(t, "2.5 kg", parsedInv.Lines[0].Item.Attributes[1].Text)
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

	require.NotNil(t, doc.Transaction.Lines[0].Agreement.ItemSellerParty)
	require.NotNil(t, doc.Transaction.Lines[0].Agreement.ItemSellerParty.GlobalID)
	assert.Equal(t, "0088", doc.Transaction.Lines[0].Agreement.ItemSellerParty.GlobalID.SchemeID)
	assert.Equal(t, "1234567890128", doc.Transaction.Lines[0].Agreement.ItemSellerParty.GlobalID.Value)

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
