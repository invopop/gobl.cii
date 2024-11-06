package ctog

import (
	"testing"

	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGCharges(t *testing.T) {
	// Invoice with Charge
	t.Run("CII_example3.xml", func(t *testing.T) {
		doc, err := loadTestXMLDoc("CII_example3.xml")
		require.NoError(t, err)

		c := NewConverter()
		err = c.NewInvoice(doc)
		require.NoError(t, err)

		inv := c.GetInvoice()

		charges := inv.Charges
		discounts := inv.Discounts
		require.NotEmpty(t, charges)

		// Check if there's a charge in the parsed output
		require.Len(t, charges, 1)
		require.Len(t, discounts, 0)
		charge := charges[0]

		assert.Equal(t, num.MakeAmount(100, 0), charge.Amount)
		assert.Equal(t, "FC", charge.Code)
		assert.Equal(t, "Freight charge", charge.Reason)

	})
	// Invoice with Discount and Charge
	t.Run("CII_business_example_02.xml", func(t *testing.T) {
		doc, err := loadTestXMLDoc("CII_business_example_02.xml")
		require.NoError(t, err)

		c := NewConverter()
		err = c.NewInvoice(doc)
		require.NoError(t, err)

		inv := c.GetInvoice()

		charges := inv.Charges
		discounts := inv.Discounts

		// Check if there's a discount in the parsed output
		require.Len(t, discounts, 1)
		require.Len(t, charges, 0)

		discount := discounts[0]

		assert.Equal(t, num.MakeAmount(0, 2), discount.Amount)
		assert.Equal(t, "Rabatt", discount.Reason)
		assert.Equal(t, "VAT", discount.Taxes[0].Category.String())
		assert.Equal(t, "standard", discount.Taxes[0].Rate.String())
		percent, err := num.PercentageFromString("19.00%")
		require.NoError(t, err)
		assert.Equal(t, &percent, discount.Taxes[0].Percent)
	})

	// Invoice with Discount and Charge
	t.Run("CII_example2.xml", func(t *testing.T) {
		doc, err := loadTestXMLDoc("CII_example2.xml")
		require.NoError(t, err)

		c := NewConverter()
		err = c.NewInvoice(doc)
		require.NoError(t, err)

		inv := c.GetInvoice()

		charges := inv.Charges
		discounts := inv.Discounts

		require.Len(t, charges, 1)
		require.Len(t, discounts, 1)

		discount := discounts[0]
		assert.Equal(t, num.MakeAmount(100, 0), discount.Amount)
		assert.Equal(t, "95", discount.Code)
		assert.Equal(t, "Promotion discount", discount.Reason)
		assert.Equal(t, "VAT", discount.Taxes[0].Category.String())
		assert.Equal(t, "standard", discount.Taxes[0].Rate.String())
		percent, err := num.PercentageFromString("25%")
		require.NoError(t, err)
		assert.Equal(t, &percent, discount.Taxes[0].Percent)

		charge := charges[0]
		assert.Equal(t, num.MakeAmount(100, 0), charge.Amount)
		assert.Equal(t, "Freight", charge.Reason)
		assert.Equal(t, "VAT", charge.Taxes[0].Category.String())
		assert.Equal(t, "standard", charge.Taxes[0].Rate.String())
		assert.Equal(t, &percent, charge.Taxes[0].Percent)

	})
}
