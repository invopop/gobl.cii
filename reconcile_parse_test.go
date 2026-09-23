package cii_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseCtoGDeclaredTotals covers reconciliation against the declared
// amounts. BT-131 is mandatory and the totals are summed from it, so it decides.
func TestParseCtoGDeclaredTotals(t *testing.T) {
	// A percentage its basis (BT-137) reproduces: the line reconciles.
	t.Run("line allowance with a declared basis", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "line-allowance-base.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.False(t, inv.HasTags(tax.TagBypass))

		require.Len(t, inv.Lines, 1)
		// 200 x 10.00, less 10% of the declared 1000.00 basis.
		assert.Equal(t, "2000.00", inv.Lines[0].Sum.String())
		assert.Equal(t, "1900.00", inv.Lines[0].Total.String())
		assert.Equal(t, "1900.00", inv.Totals.Sum.String())
		assert.Equal(t, "2280.00", inv.Totals.Payable.String())
	})

	// Reproducible under no reading: the sender's figures are kept.
	t.Run("line totals that cannot be reproduced", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "line-totals-mismatch.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.True(t, inv.HasTags(tax.TagBypass))

		require.Len(t, inv.Lines, 1)
		assert.Equal(t, "1614.87", inv.Lines[0].Total.String())
		assert.Equal(t, "1614.87", inv.Totals.Sum.String())
		assert.Equal(t, "1703.69", inv.Totals.Payable.String())
	})

	// A basis quantity contradicting the declared amount is dropped.
	t.Run("basis quantity conflicting with the declared amount", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_example8.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotEmpty(t, inv.Lines)
		assert.Equal(t, "0.00880", inv.Lines[0].Item.Price.String())
		assert.Equal(t, "140.80", inv.Lines[0].Total.String())
	})
}

// TestParseCtoGZeroPriceLines covers lines stating BT-131 with no unit price,
// which price x quantity collapses to zero. Sanitised from a Factur-X EXTENDED
// bank invoice: an exempt rate with a VATEX reason beside a standard one, and
// the whole invoice prepaid.
func TestParseCtoGZeroPriceLines(t *testing.T) {
	e, err := parseInvoiceFrom(t, "CII_zero_price_lines.xml")
	require.NoError(t, err)

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.Len(t, inv.Lines, 4)

	// These two would be 0.00, giving a 0.28 invoice against a stated 19.43.
	assert.True(t, inv.HasTags(tax.TagBypass))
	assert.Equal(t, "3.36", inv.Lines[0].Total.String())
	assert.Equal(t, "15.50", inv.Lines[1].Total.String())
	assert.Equal(t, "0.14", inv.Lines[2].Total.String())
	assert.Equal(t, "0.14", inv.Lines[3].Total.String())

	assert.Equal(t, "19.14", inv.Totals.Sum.String())
	assert.Equal(t, "0.29", inv.Totals.Tax.String())
	assert.Equal(t, "19.43", inv.Totals.TotalWithTax.String())

	// BT-113 and BT-115: the invoice is settled in full.
	require.NotNil(t, inv.Totals.Advances)
	assert.Equal(t, "19.43", inv.Totals.Advances.String())
	require.NotNil(t, inv.Totals.Due)
	assert.Equal(t, "0.00", inv.Totals.Due.String())

	// One VAT category, and the exempt rate keeps its BT-121 reason.
	require.NotNil(t, inv.Totals.Taxes)
	require.Len(t, inv.Totals.Taxes.Categories, 1)
	cat := inv.Totals.Taxes.Categories[0]
	assert.Equal(t, cbc.Code("VAT"), cat.Code)
	require.Len(t, cat.Rates, 2)
	assert.Equal(t, "VATEX-FR-CGI261C-1", cat.Rates[0].Ext.Get(cef.ExtKeyVATEX).String())
	assert.Equal(t, "15.78", cat.Rates[0].Base.String())
	assert.Equal(t, "3.36", cat.Rates[1].Base.String())
	assert.Equal(t, "0.29", cat.Rates[1].Amount.String())
}
