package cii_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGCharges(t *testing.T) {
	// Invoice with Charge
	t.Run("CII_example3.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_example3.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		charges := inv.Charges
		discounts := inv.Discounts
		require.NotEmpty(t, charges)

		// Check if there's a charge in the parsed output
		require.Len(t, charges, 1)
		require.Len(t, discounts, 0)
		charge := charges[0]

		assert.Equal(t, num.MakeAmount(10000, 2), charge.Amount)
		assert.Equal(t, "Freight charge", charge.Reason)
		assert.Equal(t, "FC", charge.Ext.Get(untdid.ExtKeyCharge).String())
	})
	// Invoice with Discount and Charge
	t.Run("CII_business_example_02.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_business_example_02.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		charges := inv.Charges
		discounts := inv.Discounts

		// Check if there's a discount in the parsed output
		require.Len(t, discounts, 1)
		require.Len(t, charges, 0)

		discount := discounts[0]

		assert.Equal(t, num.MakeAmount(0, 2), discount.Amount)
		assert.Equal(t, "Rabatt", discount.Reason)
		assert.Equal(t, "VAT", discount.Taxes[0].Category.String())
		percent, err := num.PercentageFromString("19.00%")
		require.NoError(t, err)
		assert.Equal(t, &percent, discount.Taxes[0].Percent)
	})

	// Invoice with Discount and Charge
	t.Run("CII_example2.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		charges := inv.Charges
		discounts := inv.Discounts

		require.Len(t, charges, 1)
		require.Len(t, discounts, 1)

		discount := discounts[0]
		assert.Equal(t, num.MakeAmount(10000, 2), discount.Amount)
		assert.Equal(t, "95", discount.Ext.Get(untdid.ExtKeyAllowance).String())
		assert.Equal(t, "Promotion discount", discount.Reason)
		assert.Equal(t, "VAT", discount.Taxes[0].Category.String())
		percent, err := num.PercentageFromString("25%")
		require.NoError(t, err)
		assert.Equal(t, &percent, discount.Taxes[0].Percent)

		charge := charges[0]
		assert.Equal(t, num.MakeAmount(10000, 2), charge.Amount)
		assert.Equal(t, "Freight", charge.Reason)
		assert.Equal(t, "VAT", charge.Taxes[0].Category.String())
		assert.Equal(t, &percent, charge.Taxes[0].Percent)

	})
}

// TestParseCtoGLineAllowanceAmount covers BT-136/BT-141, the actual amount of a
// line allowance or charge. EN 16931 makes it the authoritative value, so a
// calculation percentage (BT-138) must never be allowed to overwrite it.
func TestParseCtoGLineAllowanceAmount(t *testing.T) {
	e, err := parseInvoiceFrom(t, "CII_line_allowance_amount.xml")
	require.NoError(t, err)
	require.NoError(t, e.Calculate())

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.Len(t, inv.Lines, 2)

	// The first line declares a percentage with no basis amount. Applying it to
	// the line sum would give 95.90, so the percentage has to go.
	line := inv.Lines[0]
	require.Len(t, line.Discounts, 1)
	assert.Equal(t, num.MakeAmount(53227, 2), line.Discounts[0].Amount)
	assert.Nil(t, line.Discounts[0].Percent)
	require.Len(t, line.Charges, 1)
	assert.Equal(t, num.MakeAmount(2187, 2), line.Charges[0].Amount)

	// The second line backs its percentage with a basis amount that reproduces
	// the actual amount, so both survive.
	line = inv.Lines[1]
	require.Len(t, line.Discounts, 1)
	assert.Equal(t, num.MakeAmount(48694, 2), line.Discounts[0].Amount)
	require.NotNil(t, line.Discounts[0].Percent)
	assert.Equal(t, "5.92%", line.Discounts[0].Percent.String())
	require.NotNil(t, line.Discounts[0].Base)
	assert.Equal(t, num.MakeAmount(822528, 2), *line.Discounts[0].Base)

	// Neither line reproduces its declared amount (BT-131) from the terms it
	// carries, so the document keeps the sender's own figures untouched.
	assert.True(t, inv.HasTags(tax.TagBypass))
	assert.Equal(t, "1614.87", inv.Lines[0].Total.String())
	assert.Equal(t, "8137.15", inv.Lines[1].Total.String())
	assert.Equal(t, "9752.02", inv.Totals.Sum.String())
	assert.Equal(t, "536.36", inv.Totals.Tax.String())
	assert.Equal(t, "10288.38", inv.Totals.Payable.String())
}
