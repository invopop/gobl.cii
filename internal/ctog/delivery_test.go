package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl.cii/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGDelivery(t *testing.T) {
	t.Run("CII_example4.xml", func(t *testing.T) {
		doc, err := test.LoadTestXMLDoc("CII_example4.xml")
		require.NoError(t, err)

		goblEnv, err := cii.NewGOBLFromCII(doc)
		require.NoError(t, err)

		invoice, ok := goblEnv.Extract().(*bill.Invoice)
		require.True(t, ok, "Document should be an invoice")

		require.NotNil(t, invoice.Delivery, "Delivery should not be nil for Example 4")
		require.NotNil(t, invoice.Delivery.Receiver, "Delivery receiver should not be nil")

		assert.NotEmpty(t, invoice.Delivery.Receiver.Addresses, "Delivery receiver addresses should not be empty")
		assert.Equal(t, "9000", invoice.Delivery.Receiver.Addresses[0].Code, "Delivery receiver post code should match")
		assert.Equal(t, "Deliverystreet", invoice.Delivery.Receiver.Addresses[0].Street, "Delivery receiver street should match")
		assert.Equal(t, "Deliverycity", invoice.Delivery.Receiver.Addresses[0].Locality, "Delivery receiver city should match")
		assert.Equal(t, l10n.ISOCountryCode("DK"), invoice.Delivery.Receiver.Addresses[0].Country, "Delivery receiver country should match")
		assert.Equal(t, "2013-04-15", invoice.Delivery.Date.String(), "Delivery date should match")
	})

	t.Run("CII_example8.xml", func(t *testing.T) {
		doc, err := test.LoadTestXMLDoc("CII_example8.xml")
		require.NoError(t, err)

		goblEnv, err := cii.NewGOBLFromCII(doc)
		require.NoError(t, err)

		invoice, ok := goblEnv.Extract().(*bill.Invoice)
		require.True(t, ok, "Document should be an invoice")

		require.NotNil(t, invoice.Delivery, "Delivery should not be nil for Example 8")
		require.NotNil(t, invoice.Delivery.Receiver, "Delivery receiver should not be nil")
		assert.NotEmpty(t, invoice.Delivery.Receiver.Addresses, "Delivery receiver addresses should not be empty")
		assert.Equal(t, "Bedrijfslaan 4,", invoice.Delivery.Receiver.Addresses[0].Street, "Delivery receiver street should match")
		assert.Equal(t, "9999 XX", invoice.Delivery.Receiver.Addresses[0].Code, "Delivery receiver post code should match")
		assert.Equal(t, "ONDERNEMERSTAD", invoice.Delivery.Receiver.Addresses[0].Locality, "Delivery receiver city should match")
		assert.Equal(t, l10n.ISOCountryCode("NL"), invoice.Delivery.Receiver.Addresses[0].Country, "Delivery receiver country should match")
	})

}
