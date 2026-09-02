package cii_test

import (
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderingIssuer(t *testing.T) {
	// issuerEnv loads a complete invoice and attaches an ordering issuer.
	issuerEnv := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadEnvelope(t, "en16931/invoice-de-de.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		if inv.Ordering == nil {
			inv.Ordering = &bill.Ordering{}
		}
		inv.Ordering.Issuer = &org.Party{
			Name: "Billing Service Provider SL",
		}
		require.NoError(t, env.Calculate())
		return env
	}

	t.Run("maps ordering issuer to InvoicerTradeParty", func(t *testing.T) {
		doc, err := cii.ConvertInvoice(issuerEnv(t))
		require.NoError(t, err)

		invoicer := doc.Transaction.Settlement.Invoicer
		require.NotNil(t, invoicer, "Invoicer should be set from ordering.issuer")
		assert.Equal(t, "Billing Service Provider SL", invoicer.Name)
	})

	t.Run("round-trips issuer back to GOBL ordering", func(t *testing.T) {
		doc, err := cii.ConvertInvoice(issuerEnv(t))
		require.NoError(t, err)
		data, err := doc.Bytes()
		require.NoError(t, err)

		env, err := cii.Parse(data)
		require.NoError(t, err)
		out, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, out.Ordering)
		require.NotNil(t, out.Ordering.Issuer)
		assert.Equal(t, "Billing Service Provider SL", out.Ordering.Issuer.Name)
	})
}

func TestOrderingSeller(t *testing.T) {
	// invoice-de-de.json already carries the party liable for the tax in
	// ordering.seller, which CII writes as the BG-11 tax representative.
	t.Run("maps ordering seller to SellerTaxRepresentativeTradeParty", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-de-de.json")
		require.NoError(t, err)

		agmt := doc.Transaction.Agreement
		// The supplier keeps the BG-4 seller position.
		require.NotNil(t, agmt.Seller)
		assert.Equal(t, "Provide One GmbH", agmt.Seller.Name)

		rep := agmt.TaxRepresentative
		require.NotNil(t, rep, "TaxRepresentative should be set from ordering.seller")
		assert.Equal(t, "Salescompany ltd.", rep.Name)
		require.Len(t, rep.SpecifiedTaxRegistration, 1)
		assert.Equal(t, "NO923456783MVA", rep.SpecifiedTaxRegistration[0].ID.Value)
	})

	t.Run("round-trips seller back to GOBL ordering", func(t *testing.T) {
		doc, err := newInvoiceFrom(t, "en16931/invoice-de-de.json")
		require.NoError(t, err)
		data, err := doc.Bytes()
		require.NoError(t, err)

		env, err := cii.Parse(data)
		require.NoError(t, err)
		out, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, out.Supplier)
		assert.Equal(t, "Provide One GmbH", out.Supplier.Name)
		require.Len(t, out.Supplier.Inboxes, 1)
		assert.Equal(t, cbc.Code("0007"), out.Supplier.Inboxes[0].Scheme)
		assert.Equal(t, cbc.Code("111111125"), out.Supplier.Inboxes[0].Code)

		require.NotNil(t, out.Ordering)
		require.NotNil(t, out.Ordering.Seller)
		assert.Equal(t, "Salescompany ltd.", out.Ordering.Seller.Name)
		require.NotNil(t, out.Ordering.Seller.TaxID)
		assert.Equal(t, cbc.Code("923456783MVA"), out.Ordering.Seller.TaxID.Code)
	})
}

func TestOrderingCost(t *testing.T) {
	// costEnv loads a complete invoice and attaches a buyer accounting reference (BT-19).
	costEnv := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadEnvelope(t, "en16931/invoice-de-de.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		if inv.Ordering == nil {
			inv.Ordering = &bill.Ordering{}
		}
		inv.Ordering.Cost = "1287:65464"
		require.NoError(t, env.Calculate())
		return env
	}

	t.Run("maps ordering cost to ReceivableSpecifiedTradeAccountingAccount", func(t *testing.T) {
		doc, err := cii.ConvertInvoice(costEnv(t))
		require.NoError(t, err)

		acc := doc.Transaction.Settlement.AccountingAccount
		require.NotNil(t, acc, "AccountingAccount should be set from ordering.cost")
		assert.Equal(t, "1287:65464", acc.ID)
	})

	t.Run("round-trips accounting cost back to GOBL ordering", func(t *testing.T) {
		doc, err := cii.ConvertInvoice(costEnv(t))
		require.NoError(t, err)
		data, err := doc.Bytes()
		require.NoError(t, err)

		env, err := cii.Parse(data)
		require.NoError(t, err)
		out, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, out.Ordering)
		assert.Equal(t, cbc.Code("1287:65464"), out.Ordering.Cost)
	})
}
