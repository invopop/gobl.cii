package cii_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseParty function
func TestParseCtoGParty(t *testing.T) {
	t.Run("invoice-test-01.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "invoice-test-01.xml")
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
		e, err := parseInvoiceFrom(t, "CII_example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		supplier := inv.Supplier
		require.NotNil(t, supplier)

		assert.Equal(t, "Salescompany ltd.", supplier.Name)
		require.NotNil(t, supplier.TaxID)
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

		// BG-11 tax representative, the party liable for the tax.
		seller := inv.Ordering.Seller
		require.NotNil(t, seller)

		assert.Equal(t, "Tax handling company AS", seller.Name)
		require.NotNil(t, seller.TaxID)
		assert.Equal(t, cbc.Code("967611265MVA"), seller.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), seller.TaxID.Country)
		require.Len(t, seller.Addresses, 1)
		assert.Equal(t, "Regent street", seller.Addresses[0].Street)
		assert.Equal(t, "Newtown", seller.Addresses[0].Locality)
		assert.Equal(t, cbc.Code("202"), seller.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), seller.Addresses[0].Country)

		customer := inv.Customer
		require.NotNil(t, customer)

		assert.Equal(t, "The Buyercompany", customer.Name)
		assert.Equal(t, cbc.Code("987654321MVA"), customer.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), customer.TaxID.Country)
		require.Len(t, customer.Identities, 2)
		// BT-47: Legal registration identifier
		assert.Equal(t, cbc.Code("987654321"), customer.Identities[0].Code)
		assert.Equal(t, org.IdentityScopeLegal, customer.Identities[0].Scope)
		// GlobalID
		assert.Equal(t, "3456789012098", customer.Identities[1].Code.String())
		assert.Equal(t, "0088", customer.Identities[1].Ext.Get(iso.ExtKeySchemeID).String())
	})

	t.Run("CII-IN_SE-R-003.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII-IN_SE-R-003.xml")
		require.NoError(t, err)
		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		supplier := inv.Supplier
		require.NotNil(t, supplier)

		assert.Equal(t, "5566778899", supplier.Inboxes[0].Code.String())
		assert.Equal(t, "0007", supplier.Inboxes[0].Scheme.String())

		// BT-31: the fixture spells the scheme "VAT" instead of "VA",
		// which must still be read as the VAT identifier.
		require.NotNil(t, supplier.TaxID)
		assert.Equal(t, l10n.TaxCountryCode("SE"), supplier.TaxID.Country)
		assert.Equal(t, cbc.Code("123456789001"), supplier.TaxID.Code)

		// BT-32: an "FC" registration keeps the tax scope on the way in
		var taxIDs []*org.Identity
		for _, id := range supplier.Identities {
			if id.Scope == org.IdentityScopeTax {
				taxIDs = append(taxIDs, id)
			}
		}
		require.Len(t, taxIDs, 1)
		assert.Equal(t, l10n.ISOCountryCode("SE"), taxIDs[0].Country)
		assert.Contains(t, taxIDs[0].Code.String(), "F-skatt")
	})
}

// TestParseCtoGSupplierTaxRepresentative checks that the supplier keeps its own
// BT-31 VAT number and BT-34 endpoint whether or not the invoice names a BG-11
// tax representative. The two fixtures are the same invoice, one with the
// SellerTaxRepresentativeTradeParty and one without.
func TestParseCtoGSupplierTaxRepresentative(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		repName  string   // BT-62, empty when there is no tax representative
		repTaxID cbc.Code // BT-63
	}{
		{
			name: "without tax representative",
			file: "CII_example2_no_tax_representative.xml",
		},
		{
			name:     "with tax representative",
			file:     "CII_example2.xml",
			repName:  "Tax handling company AS",
			repTaxID: "967611265MVA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := parseInvoiceFrom(t, tt.file)
			require.NoError(t, err)

			inv, ok := e.Extract().(*bill.Invoice)
			require.True(t, ok)

			supplier := inv.Supplier
			require.NotNil(t, supplier)
			assert.Equal(t, "Salescompany ltd.", supplier.Name)
			require.NotNil(t, supplier.TaxID)
			assert.Equal(t, l10n.TaxCountryCode("NO"), supplier.TaxID.Country)
			assert.Equal(t, cbc.Code("123456789MVA"), supplier.TaxID.Code)
			require.Len(t, supplier.Inboxes, 1)
			assert.Equal(t, "inbox@example.com", supplier.Inboxes[0].Email)

			var rep *org.Party
			if inv.Ordering != nil {
				rep = inv.Ordering.Seller
			}
			if tt.repName == "" {
				assert.Nil(t, rep)
				return
			}
			require.NotNil(t, rep)
			assert.Equal(t, tt.repName, rep.Name)
			require.NotNil(t, rep.TaxID)
			assert.Equal(t, tt.repTaxID, rep.TaxID.Code)
		})
	}
}
