package ctog

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGCharges(t *testing.T) {
	// Invoice with Charge
	t.Run("CII_example3.xml", func(t *testing.T) {
		e, err := newDocumentFrom("CII_example3.xml")
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
		assert.Equal(t, "FC", charge.Ext[untdid.ExtKeyCharge].String())
	})
	// Invoice with Discount and Charge
	t.Run("CII_business_example_02.xml", func(t *testing.T) {
		e, err := newDocumentFrom("CII_business_example_02.xml")
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
		e, err := newDocumentFrom("CII_example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		charges := inv.Charges
		discounts := inv.Discounts

		require.Len(t, charges, 1)
		require.Len(t, discounts, 1)

		discount := discounts[0]
		assert.Equal(t, num.MakeAmount(10000, 2), discount.Amount)
		assert.Equal(t, "95", discount.Ext[untdid.ExtKeyAllowance].String())
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
