package gtoc

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgreement(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := NewDocumentFrom("invoice-de-de.json")
		require.NoError(t, err)

		assert.Nil(t, err)
		assert.Equal(t, "XR-2024-2", doc.Transaction.Agreement.BuyerReference)
		assert.Equal(t, "Provide One GmbH", doc.Transaction.Agreement.TaxRepresentative.Name)
		assert.Equal(t, "John Doe", doc.Transaction.Agreement.TaxRepresentative.Contact.PersonName)
		assert.Equal(t, "+49100200300", doc.Transaction.Agreement.TaxRepresentative.Contact.Phone.CompleteNumber)
		assert.Equal(t, "billing@example.com", doc.Transaction.Agreement.TaxRepresentative.URIUniversalCommunication.URIID)
		assert.Equal(t, "69190", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.Postcode)
		assert.Equal(t, "Dietmar-Hopp-Allee", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.LineOne)
		assert.Equal(t, "16", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.LineTwo)
		assert.Equal(t, "Walldorf", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.City)
		assert.Equal(t, "DE", doc.Transaction.Agreement.TaxRepresentative.PostalTradeAddress.CountryID)
	})

	t.Run("invoice-complete.json", func(t *testing.T) {
		env, err := LoadTestEnvelope("invoice-complete.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		converter := NewConverter()
		err = converter.newDocument(inv)
		require.NoError(t, err)

		doc := converter.GetDocument()
		assert.Nil(t, err)
		assert.Equal(t, "PO4711", doc.Transaction.Agreement.BuyerReference)
		assert.Equal(t, "2013-05", doc.Transaction.Agreement.Contract)
	})
}
