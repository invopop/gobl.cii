package ctog

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseParty function
func TestParseCtoGParty(t *testing.T) {
	t.Run("invoice-test-01.xml", func(t *testing.T) {
		e, err := newDocumentFrom("invoice-test-01.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		seller := inv.Supplier
		buyer := inv.Customer
		require.NotNil(t, seller)

		assert.Equal(t, "Sample Seller", seller.Name)
		assert.Equal(t, l10n.TaxCountryCode("DE"), seller.TaxID.Country)
		assert.Equal(t, cbc.Code("049120826"), seller.TaxID.Code)

		assert.Equal(t, "Sample Buyer", buyer.Name)
		assert.Equal(t, "Sample Street 2", buyer.Addresses[0].Street)
		assert.Equal(t, "Sample City", buyer.Addresses[0].Locality)
		assert.Equal(t, cbc.Code("48000"), buyer.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("DE"), buyer.Addresses[0].Country)
	})

	// With SellerTaxRepresentativeTradeParty
	t.Run("CII_example2.xml", func(t *testing.T) {
		e, err := newDocumentFrom("CII_example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		seller := inv.Supplier
		require.NotNil(t, seller)

		assert.NotNil(t, seller.TaxID)
		assert.Equal(t, cbc.Code("967611265MVA"), seller.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), seller.TaxID.Country)

		assert.Equal(t, "Tax handling company AS", seller.Name)
		require.Len(t, seller.Addresses, 1)
		assert.Equal(t, "Regent street", seller.Addresses[0].Street)
		assert.Equal(t, "Newtown", seller.Addresses[0].Locality)
		assert.Equal(t, cbc.Code("202"), seller.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), seller.Addresses[0].Country)

		// Test parsing of supplier
		supplier := inv.Ordering.Seller
		require.NotNil(t, supplier)

		assert.Equal(t, "Salescompany ltd.", supplier.Name)
		assert.Equal(t, cbc.Code("123456789MVA"), supplier.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), supplier.TaxID.Country)
		assert.Equal(t, "inbox@example.com", supplier.Inboxes[0].Email)

		require.Len(t, supplier.Addresses, 1)
		assert.Equal(t, "Main street 34", supplier.Addresses[0].Street)
		assert.Equal(t, "Suite 123", supplier.Addresses[0].StreetExtra)
		assert.Equal(t, "Big city", supplier.Addresses[0].Locality)
		assert.Equal(t, "RegionA", supplier.Addresses[0].Region)
		assert.Equal(t, cbc.Code("303"), supplier.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), supplier.Addresses[0].Country)
		require.Len(t, supplier.People, 1)
		assert.Equal(t, "Antonio Salesmacher", supplier.People[0].Name.Given)
		require.Len(t, supplier.Emails, 1)
		assert.Equal(t, "antonio@salescompany.no", supplier.Emails[0].Address)
		require.Len(t, supplier.Telephones, 1)
		assert.Equal(t, "46211230", supplier.Telephones[0].Number)

		customer := inv.Customer
		require.NotNil(t, customer)

		assert.Equal(t, "The Buyercompany", customer.Name)
		assert.Equal(t, cbc.Code("987654321MVA"), customer.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), customer.TaxID.Country)
		require.Len(t, customer.Identities, 1)
		assert.Equal(t, "3456789012098", customer.Identities[0].Code.String())
		assert.Equal(t, "0088", customer.Identities[0].Ext[iso.ExtKeySchemeID].String())
	})
}
