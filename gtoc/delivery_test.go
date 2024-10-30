package gtoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDelivery(t *testing.T) {
	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := NewDocumentFrom("invoice-complete.json")
		require.NoError(t, err)

		assert.Equal(t, "20240210", doc.Transaction.Delivery.Event.Date)
		assert.NotNil(t, doc.Transaction.Delivery.Receiver)
		assert.NotNil(t, doc.Transaction.Delivery.Receiver.PostalTradeAddress)
		assert.Equal(t, "Deliverystreet 2", doc.Transaction.Delivery.Receiver.PostalTradeAddress.LineOne)
		assert.Equal(t, "DeliveryCity", doc.Transaction.Delivery.Receiver.PostalTradeAddress.City)
		assert.Equal(t, "523427", doc.Transaction.Delivery.Receiver.PostalTradeAddress.Postcode)
		assert.Equal(t, "RegionD", doc.Transaction.Delivery.Receiver.PostalTradeAddress.Region)
		assert.Equal(t, "NO", doc.Transaction.Delivery.Receiver.PostalTradeAddress.CountryID)

		assert.NotNil(t, doc.Transaction.Delivery)
		assert.Equal(t, "3544", *doc.Transaction.Delivery.Receiving)
		assert.Equal(t, "5433", *doc.Transaction.Delivery.Despatch)
	})
}
