package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAllowanceCharges(t *testing.T) {
	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-complete.json")
		require.NoError(t, err)
		// Document Level
		assert.Len(t, doc.Transaction.Settlement.AllowanceCharges, 2)

		assert.True(t, doc.Transaction.Settlement.AllowanceCharges[0].ChargeIndicator.Value)
		assert.Equal(t, "11.00", doc.Transaction.Settlement.AllowanceCharges[0].Amount)
		assert.Equal(t, "Freight", doc.Transaction.Settlement.AllowanceCharges[0].Reason)

		assert.False(t, doc.Transaction.Settlement.AllowanceCharges[1].ChargeIndicator.Value)
		assert.Equal(t, "88", doc.Transaction.Settlement.AllowanceCharges[1].ReasonCode)
		assert.Equal(t, "10.00", doc.Transaction.Settlement.AllowanceCharges[1].Amount)
		assert.Equal(t, "Promotion discount", doc.Transaction.Settlement.AllowanceCharges[1].Reason)
	})

	t.Run("discount exemption reason matches header category", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-exempt.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Discounts = append(inv.Discounts, &bill.Discount{
			Reason: "Loyalty discount",
			Amount: num.MakeAmount(500, 2),
			Taxes:  tax.Set{inv.Lines[0].Taxes[0]},
		})
		require.NoError(t, env.Calculate())

		doc, err := cii.ConvertInvoice(env)
		require.NoError(t, err)

		require.NotEmpty(t, doc.Transaction.Settlement.AllowanceCharges)
		ac := doc.Transaction.Settlement.AllowanceCharges[len(doc.Transaction.Settlement.AllowanceCharges)-1]
		require.NotNil(t, ac.Tax)
		assert.Equal(t, "VATEX-EU-132", ac.Tax.ExemptionReasonCode)

		// Line-level ApplicableTradeTax must never carry ExemptionReasonCode:
		// at least the Factur-X profile marks it as not used in that context.
		for _, line := range doc.Transaction.Lines {
			for _, lineTax := range line.TradeSettlement.ApplicableTradeTax {
				assert.Empty(t, lineTax.ExemptionReasonCode)
			}
		}
	})

	t.Run("invoice-without-buyers-tax-id.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		//Line Level
		assert.Len(t, doc.Transaction.Lines, 1)
		assert.Len(t, doc.Transaction.Lines[0].TradeSettlement.AllowanceCharge, 2)
		assert.True(t, doc.Transaction.Lines[0].TradeSettlement.AllowanceCharge[0].ChargeIndicator.Value)
		assert.Equal(t, "Testing", doc.Transaction.Lines[0].TradeSettlement.AllowanceCharge[0].Reason)
		assert.Equal(t, "12.00", doc.Transaction.Lines[0].TradeSettlement.AllowanceCharge[0].Amount)
		assert.False(t, doc.Transaction.Lines[0].TradeSettlement.AllowanceCharge[1].ChargeIndicator.Value)
		assert.Equal(t, "Damage", doc.Transaction.Lines[0].TradeSettlement.AllowanceCharge[1].Reason)
		assert.Equal(t, "12.00", doc.Transaction.Lines[0].TradeSettlement.AllowanceCharge[1].Amount)

	})
}
