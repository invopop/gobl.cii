package ctog

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGDelivery(t *testing.T) {
	t.Run("CII_example4.xml", func(t *testing.T) {
		e, err := newDocumentFrom("CII_example4.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		delivery := inv.Delivery

		require.NotNil(t, delivery, "Delivery should not be nil for Example 4")
		require.NotNil(t, delivery.Receiver, "Delivery receiver should not be nil")

		assert.NotEmpty(t, delivery.Receiver.Addresses, "Delivery receiver addresses should not be empty")
		assert.Equal(t, cbc.Code("9000"), delivery.Receiver.Addresses[0].Code, "Delivery receiver post code should match")
		assert.Equal(t, "Deliverystreet", delivery.Receiver.Addresses[0].Street, "Delivery receiver street should match")
		assert.Equal(t, "Deliverycity", delivery.Receiver.Addresses[0].Locality, "Delivery receiver city should match")
		assert.Equal(t, l10n.ISOCountryCode("DK"), delivery.Receiver.Addresses[0].Country, "Delivery receiver country should match")
		assert.Equal(t, "2013-04-15", delivery.Date.String(), "Delivery date should match")
	})

	t.Run("CII_example8.xml", func(t *testing.T) {
		e, err := newDocumentFrom("CII_example8.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		delivery := inv.Delivery

		require.NotNil(t, delivery, "Delivery should not be nil for Example 8")
		require.NotNil(t, delivery.Receiver, "Delivery receiver should not be nil")
		assert.NotEmpty(t, delivery.Receiver.Addresses, "Delivery receiver addresses should not be empty")
		assert.Equal(t, "Bedrijfslaan 4,", delivery.Receiver.Addresses[0].Street, "Delivery receiver street should match")
		assert.Equal(t, cbc.Code("9999 XX"), delivery.Receiver.Addresses[0].Code, "Delivery receiver post code should match")
		assert.Equal(t, "ONDERNEMERSTAD", delivery.Receiver.Addresses[0].Locality, "Delivery receiver city should match")
		assert.Equal(t, l10n.ISOCountryCode("NL"), delivery.Receiver.Addresses[0].Country, "Delivery receiver country should match")
	})

}
