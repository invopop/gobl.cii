package cii_test

import (
	"fmt"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSeller(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		// Example With Tax Rep
		assert.Equal(t, "Provide One GmbH", doc.Transaction.Agreement.TaxRepresentative.Name)
		assert.Equal(t, "John Doe", doc.Transaction.Agreement.TaxRepresentative.Contact.PersonName)
		assert.Equal(t, "+49100200300", doc.Transaction.Agreement.TaxRepresentative.Contact.Phone.CompleteNumber)
		assert.Equal(t, "69190", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.Postcode)
		assert.Equal(t, "Dietmar-Hopp-Allee 16", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.LineOne)
		assert.Equal(t, "Walldorf", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.City)
		assert.Equal(t, "DE", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.CountryID)
		assert.Equal(t, "billing@example.com", doc.Transaction.Agreement.TaxRepresentative.Contact.Email.URIID)

		assert.Equal(t, "Salescompany ltd.", doc.Transaction.Agreement.Seller.Name)
		assert.Equal(t, "Antonio Salesmacher", doc.Transaction.Agreement.Seller.Contact.PersonName)
		assert.Equal(t, "46211230", doc.Transaction.Agreement.Seller.Contact.Phone.CompleteNumber)
		assert.Equal(t, "303", doc.Transaction.Agreement.Seller.PostalTradeAddress.Postcode)
		assert.Equal(t, "Main street 34", doc.Transaction.Agreement.Seller.PostalTradeAddress.LineOne)
		assert.Equal(t, "Big city", doc.Transaction.Agreement.Seller.PostalTradeAddress.City)
		assert.Equal(t, "NO", doc.Transaction.Agreement.Seller.PostalTradeAddress.CountryID)
		assert.Equal(t, "antonio@salescompany.no", doc.Transaction.Agreement.Seller.Contact.Email.URIID)
		assert.Equal(t, "NO123456789MVA", doc.Transaction.Agreement.Seller.SpecifiedTaxRegistration[0].ID.Value)
		assert.Equal(t, "inbox@example.com", doc.Transaction.Agreement.Seller.URIUniversalCommunication.ID.Value)
		assert.Equal(t, cii.SchemeIDEmail, doc.Transaction.Agreement.Seller.URIUniversalCommunication.ID.SchemeID)

		assert.Equal(t, "Sample Consumer", doc.Transaction.Agreement.Buyer.Name)
		assert.Equal(t, "80939", doc.Transaction.Agreement.Buyer.PostalTradeAddress.Postcode)
		assert.Equal(t, "Werner-Heisenberg-Allee 25", doc.Transaction.Agreement.Buyer.PostalTradeAddress.LineOne)
		assert.Equal(t, "München", doc.Transaction.Agreement.Buyer.PostalTradeAddress.City)
		assert.Equal(t, "DE", doc.Transaction.Agreement.Buyer.PostalTradeAddress.CountryID)
		fmt.Println(doc.Transaction.Agreement.Buyer.Contact)
		assert.Equal(t, "email@sample.com", doc.Transaction.Agreement.Buyer.Contact.Email.URIID)
		assert.Equal(t, "DE282741168", doc.Transaction.Agreement.Buyer.SpecifiedTaxRegistration[0].ID.Value)

		assert.Equal(t, "123456789", doc.Transaction.Agreement.Buyer.GlobalID.Value)
		assert.Equal(t, "0088", doc.Transaction.Agreement.Buyer.GlobalID.SchemeID)
	})
}
