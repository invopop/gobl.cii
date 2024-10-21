package ctog

import (
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseParty function
func TestParseCtoGParty(t *testing.T) {
	t.Run("invoice-test-01.xml", func(t *testing.T) {
		xmlData, err := LoadTestXMLDoc("invoice-test-01.xml")
		require.NoError(t, err)

		c := NewConversor()
		inv, err := c.NewInvoice(xmlData)
		require.NoError(t, err)

		seller := inv.Supplier
		buyer := inv.Customer
		require.NotNil(t, seller)

		assert.Equal(t, "Sample Seller", seller.Name)
		assert.Equal(t, l10n.TaxCountryCode("DE"), seller.TaxID.Country)
		assert.Equal(t, cbc.Code("049120826"), seller.TaxID.Code)

		assert.Equal(t, "Sample Buyer", buyer.Name)
		assert.Equal(t, "Sample Street 2", buyer.Addresses[0].Street)
		assert.Equal(t, "Sample City", buyer.Addresses[0].Locality)
		assert.Equal(t, "48000", buyer.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("DE"), buyer.Addresses[0].Country)
	})

	// With SellerTaxRepresentativeTradeParty
	t.Run("CII_example2.xml", func(t *testing.T) {
		xmlData, err := LoadTestXMLDoc("CII_example2.xml")
		require.NoError(t, err)

		c := NewConversor()
		inv, err := c.NewInvoice(xmlData)
		require.NoError(t, err)

		taxRep := inv.Supplier
		require.NotNil(t, taxRep)

		assert.NotNil(t, taxRep.TaxID)
		assert.Equal(t, cbc.Code("967611265"), taxRep.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), taxRep.TaxID.Country)

		assert.Equal(t, "Tax handling company AS", taxRep.Name)
		require.Len(t, taxRep.Addresses, 1)
		assert.Equal(t, "Regent street", taxRep.Addresses[0].Street)
		assert.Equal(t, "Newtown", taxRep.Addresses[0].Locality)
		assert.Equal(t, "202", taxRep.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), taxRep.Addresses[0].Country)

		// Test parsing of supplier
		supplier := inv.Supplier
		require.NotNil(t, supplier)

		assert.Equal(t, "Salescompany ltd.", supplier.Name)
		assert.Equal(t, cbc.Code("123456789"), supplier.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), supplier.TaxID.Country)

		require.Len(t, supplier.Addresses, 1)
		assert.Equal(t, "Main street 34", supplier.Addresses[0].Street)
		assert.Equal(t, "Suite 123", supplier.Addresses[0].StreetExtra)
		assert.Equal(t, "Big city", supplier.Addresses[0].Locality)
		assert.Equal(t, "RegionA", supplier.Addresses[0].Region)
		assert.Equal(t, "303", supplier.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), supplier.Addresses[0].Country)

		require.Len(t, supplier.People, 1)
		assert.Equal(t, "Antonio Salesmacher", supplier.People[0].Name.Given)

		require.Len(t, supplier.Emails, 1)
		assert.Equal(t, "antonio@salescompany.no", supplier.Emails[0].Address)

		require.Len(t, supplier.Telephones, 1)
		assert.Equal(t, "46211230", supplier.Telephones[0].Number)
	})
}
