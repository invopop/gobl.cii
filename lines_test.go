package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
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
