package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
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
		assert.Equal(t, "MARCHE", doc.Transaction.Agreement.Contract.ReferenceTypeCode)
	})
}

func TestContractReferenceTypeRoundTrip(t *testing.T) {
	env := loadEnvelope(t, "en16931/invoice-complete.json")
	doc, err := cii.ConvertInvoice(env)
	require.NoError(t, err)

	require.NotNil(t, doc.Transaction.Agreement.Contract)
	assert.Equal(t, "MARCHE", doc.Transaction.Agreement.Contract.ReferenceTypeCode)

	data, err := doc.Bytes()
	require.NoError(t, err)

	parsed, err := cii.Parse(data)
	require.NoError(t, err)
	parsedInv, ok := parsed.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.NotEmpty(t, parsedInv.Ordering.Contracts)
	assert.Equal(t, "MARCHE", parsedInv.Ordering.Contracts[0].Reason)
}
