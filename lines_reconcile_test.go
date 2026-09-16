package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLineAllowanceBase(t *testing.T) {
	e, err := parseInvoiceFrom(t, "line-allowance-base.xml")
	require.NoError(t, err)
	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.Len(t, inv.Lines, 1)
	require.Len(t, inv.Lines[0].Discounts, 1)
	d := inv.Lines[0].Discounts[0]

	// BT-138 applies to BT-137, not to the line sum of 2000.00.
	require.NotNil(t, d.Base)
	assert.Equal(t, "1000.00", d.Base.String())
	assert.Equal(t, "10.00%", d.Percent.String())
	assert.Equal(t, "100.00", d.Amount.String())
	assert.Equal(t, "1900.00", inv.Lines[0].Total.String())
	assert.Equal(t, "2280.00", inv.Totals.Payable.String())
	assert.Empty(t, inv.Lines[0].Notes)
}

func TestParseLineTotalsMismatch(t *testing.T) {
	e, err := parseInvoiceFrom(t, "line-totals-mismatch.xml")
	require.NoError(t, err)
	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.Len(t, inv.Lines, 1)
	l := inv.Lines[0]

	assert.Equal(t, "0.598100", l.Item.Price.String())
	assert.Equal(t, "1614.87", l.Total.String())
	assert.Empty(t, l.Discounts)
	assert.Empty(t, l.Charges)

	assert.Equal(t, "1614.87", inv.Totals.Sum.String())
	assert.Equal(t, "88.82", inv.Totals.Tax.String())
	assert.Equal(t, "1703.69", inv.Totals.Payable.String())

	require.Len(t, l.Notes, 1)
	assert.Equal(t, cii.NoteSrcReconciliation, l.Notes[0].Src)
	assert.Contains(t, l.Notes[0].Text, "allowance of 532.27")
	assert.Contains(t, l.Notes[0].Text, "charge of 21.87")
}

// Generated notes must not re-export as the issuer's own BT-22 text.
func TestReconciliationNotesNotExported(t *testing.T) {
	e, err := parseInvoiceFrom(t, "line-totals-mismatch.xml")
	require.NoError(t, err)

	doc, err := cii.ConvertInvoice(e)
	require.NoError(t, err)
	data, err := doc.Bytes()
	require.NoError(t, err)
	assert.NotContains(t, string(data), "as received")
}
