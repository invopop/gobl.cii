package cii_test

import (
	"testing"

	ctog "github.com/invopop/gobl.cii/internal/ctog"
	"github.com/invopop/gobl.cii/test"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseParty function
func TestParseCtoGParty(t *testing.T) {
	t.Run("invoice-test-01.xml", func(t *testing.T) {
		doc, err := test.LoadTestXMLDoc("invoice-test-01.xml")
		require.NoError(t, err)

		seller := ctog.ParseCtoGParty(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTradeParty)
		buyer := ctog.ParseCtoGParty(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerTradeParty)
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
		doc, err := test.LoadTestXMLDoc("CII_example2.xml")
		require.NoError(t, err)

		party := ctog.ParseCtoGParty(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTaxRepresentativeTradeParty)
		require.NotNil(t, party)

		assert.NotNil(t, party.TaxID)
		assert.Equal(t, cbc.Code("967611265"), party.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), party.TaxID.Country)

		assert.Equal(t, "Tax handling company AS", party.Name)
		require.Len(t, party.Addresses, 1)
		assert.Equal(t, "Regent street", party.Addresses[0].Street)
		assert.Equal(t, "Newtown", party.Addresses[0].Locality)
		assert.Equal(t, "202", party.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), party.Addresses[0].Country)

		// Test parsing of ordering.seller
		orderingSeller := ctog.ParseCtoGParty(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTradeParty)
		require.NotNil(t, orderingSeller)

		assert.Equal(t, "Salescompany ltd.", orderingSeller.Name)
		assert.Equal(t, cbc.Code("123456789"), orderingSeller.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), orderingSeller.TaxID.Country)

		require.Len(t, orderingSeller.Addresses, 1)
		assert.Equal(t, "Main street 34", orderingSeller.Addresses[0].Street)
		assert.Equal(t, "Suite 123", orderingSeller.Addresses[0].StreetExtra)
		assert.Equal(t, "Big city", orderingSeller.Addresses[0].Locality)
		assert.Equal(t, "RegionA", orderingSeller.Addresses[0].Region)
		assert.Equal(t, "303", orderingSeller.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), orderingSeller.Addresses[0].Country)

		require.Len(t, orderingSeller.People, 1)
		assert.Equal(t, "Antonio Salesmacher", orderingSeller.People[0].Name.Given)

		require.Len(t, orderingSeller.Emails, 1)
		assert.Equal(t, "antonio@salescompany.no", orderingSeller.Emails[0].Address)

		require.Len(t, orderingSeller.Telephones, 1)
		assert.Equal(t, "46211230", orderingSeller.Telephones[0].Number)
	})
}
