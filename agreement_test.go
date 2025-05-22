package cii_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgreement(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "XR-2024-2", doc.Transaction.Agreement.BuyerReference)
		assert.Equal(t, "Provide One GmbH", doc.Transaction.Agreement.TaxRepresentative.Name)
		assert.Equal(t, "69190", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.Postcode)
		assert.Equal(t, "Dietmar-Hopp-Allee 16", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.LineOne)
		assert.Equal(t, "Walldorf", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.City)
		assert.Equal(t, "DE", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.CountryID)
		assert.Equal(t, "DE111111125", doc.Transaction.Agreement.TaxRepresentative.SpecifiedTaxRegistration[0].ID.Value)
	})

	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-complete.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "PO4711", doc.Transaction.Agreement.BuyerReference)
		assert.Equal(t, "2013-05", doc.Transaction.Agreement.Contract.ID)
	})
}
