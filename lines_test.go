package cii_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLines(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "invoice-de-de.json")
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
	})

}
