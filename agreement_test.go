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
		// The supplier stays the seller, ordering.seller becomes the BG-11
		// tax representative.
		assert.Equal(t, "Provide One GmbH", doc.Transaction.Agreement.Seller.Name)
		assert.Equal(t, "Salescompany ltd.", doc.Transaction.Agreement.TaxRepresentative.Name)
		assert.Equal(t, "303", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.Postcode)
		assert.Equal(t, "Main street 34", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.LineOne)
		assert.Equal(t, "Big city", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.City)
		assert.Equal(t, "NO", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.CountryID)
		assert.Equal(t, "NO923456783MVA", doc.Transaction.Agreement.TaxRepresentative.SpecifiedTaxRegistration[0].ID.Value)
		// BG-11 carries no ID (CII-SR-282 to 291).
		assert.Nil(t, doc.Transaction.Agreement.TaxRepresentative.ID)
	})

	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-complete.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "PO4711", doc.Transaction.Agreement.BuyerReference)
		assert.Equal(t, "2013-05", doc.Transaction.Agreement.Contract.ID)
	})
}
