package gtoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAllowanceCharges(t *testing.T) {
	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := NewDocumentFrom("invoice-complete.json")
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

	t.Run("invoice-without-buyers-tax-id.json", func(t *testing.T) {
		doc, err := NewDocumentFrom("invoice-without-buyers-tax-id.json")
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
