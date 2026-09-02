package cii_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSeller(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		// The supplier is the BG-4 seller.
		assert.Equal(t, "Provide One GmbH", doc.Transaction.Agreement.Seller.Name)
		assert.Equal(t, "John Doe", doc.Transaction.Agreement.Seller.Contact.PersonName)
		assert.Equal(t, "+49100200300", doc.Transaction.Agreement.Seller.Contact.Phone.CompleteNumber)
		assert.Equal(t, "69190", doc.Transaction.Agreement.Seller.PostalTradeAddress.Postcode)
		assert.Equal(t, "Dietmar-Hopp-Allee 16", doc.Transaction.Agreement.Seller.PostalTradeAddress.LineOne)
		assert.Equal(t, "Walldorf", doc.Transaction.Agreement.Seller.PostalTradeAddress.City)
		assert.Equal(t, "DE", doc.Transaction.Agreement.Seller.PostalTradeAddress.CountryID)
		assert.Equal(t, "billing@example.com", doc.Transaction.Agreement.Seller.Contact.Email.URIID)
		assert.Equal(t, "DE111111125", doc.Transaction.Agreement.Seller.SpecifiedTaxRegistration[0].ID.Value)
		assert.Equal(t, "111111125", doc.Transaction.Agreement.Seller.URIUniversalCommunication.ID.Value)
		assert.Equal(t, "0007", doc.Transaction.Agreement.Seller.URIUniversalCommunication.ID.SchemeID)

		// Example With Tax Rep: ordering.seller becomes the BG-11 party.
		assert.Equal(t, "Salescompany ltd.", doc.Transaction.Agreement.TaxRepresentative.Name)
		assert.Equal(t, "303", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.Postcode)
		assert.Equal(t, "Main street 34", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.LineOne)
		assert.Equal(t, "Big city", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.City)
		assert.Equal(t, "NO", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.CountryID)
		assert.Equal(t, "NO923456783MVA", doc.Transaction.Agreement.TaxRepresentative.SpecifiedTaxRegistration[0].ID.Value)

		assert.Equal(t, "Sample Consumer", doc.Transaction.Agreement.Buyer.Name)
		assert.Equal(t, "80939", doc.Transaction.Agreement.Buyer.PostalTradeAddress.Postcode)
		assert.Equal(t, "Werner-Heisenberg-Allee 25", doc.Transaction.Agreement.Buyer.PostalTradeAddress.LineOne)
		assert.Equal(t, "München", doc.Transaction.Agreement.Buyer.PostalTradeAddress.City)
		assert.Equal(t, "DE", doc.Transaction.Agreement.Buyer.PostalTradeAddress.CountryID)
		assert.Equal(t, "email@sample.com", doc.Transaction.Agreement.Buyer.Contact.Email.URIID)
		assert.Equal(t, "DE282741168", doc.Transaction.Agreement.Buyer.SpecifiedTaxRegistration[0].ID.Value)

		assert.Equal(t, "123456789", doc.Transaction.Agreement.Buyer.GlobalID.Value)
		assert.Equal(t, "0088", doc.Transaction.Agreement.Buyer.GlobalID.SchemeID)
	})
}
