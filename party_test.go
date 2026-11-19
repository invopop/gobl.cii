package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
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

		require.Len(t, doc.Transaction.Agreement.Buyer.GlobalID, 1)
		assert.Equal(t, "123456789", doc.Transaction.Agreement.Buyer.GlobalID[0].Value)
		assert.Equal(t, "0088", doc.Transaction.Agreement.Buyer.GlobalID[0].SchemeID)
	})
}

// TestPartyRegistration checks that BT-33 (additional legal information)
// round-trips through ram:Description for both the seller (supplier) and
// the buyer (customer).
func TestPartyRegistration(t *testing.T) {
	env := loadEnvelope(t, "en16931/invoice-complete.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	inv.Supplier.Registration = &org.Registration{Other: "Share capital 100.000 EUR"}
	inv.Customer.Registration = &org.Registration{Other: "GmbH"}
	require.NoError(t, env.Calculate())

	// GOBL -> CII
	doc, err := cii.ExportInvoice(env)
	require.NoError(t, err)
	assert.Equal(t, "Share capital 100.000 EUR", doc.Transaction.Agreement.Seller.Description)
	assert.Equal(t, "GmbH", doc.Transaction.Agreement.Buyer.Description)

	// CII -> GOBL
	data, err := cii.Encode(doc)
	require.NoError(t, err)
	back, err := parseCII(data)
	require.NoError(t, err)
	inv2, ok := back.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.NotNil(t, inv2.Supplier.Registration)
	assert.Equal(t, "Share capital 100.000 EUR", inv2.Supplier.Registration.Other)
	require.NotNil(t, inv2.Customer.Registration)
	assert.Equal(t, "GmbH", inv2.Customer.Registration.Other)
}
