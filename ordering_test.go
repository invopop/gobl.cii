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

func TestExtendedParties(t *testing.T) {
	const fixture = "peppol-france-extended/invoice-extended-parties.json"

	convert := func(t *testing.T, ctx cii.Context) *cii.Invoice {
		t.Helper()
		doc, err := cii.ConvertInvoice(loadEnvelope(t, fixture), cii.WithContext(ctx))
		require.NoError(t, err)
		return doc
	}

	t.Run("french extended maps the facturant, the addressee and the payer", func(t *testing.T) {
		stlm := convert(t, cii.ContextPeppolFranceExtendedV1).Transaction.Settlement

		// EXT-FR-FE-BG-05, pinned to UNCL 3035 "II" by EXT-FR-FE-113.
		require.NotNil(t, stlm.Invoicer)
		assert.Equal(t, "Facturant SARL", stlm.Invoicer.Name)
		assert.Equal(t, "II", stlm.Invoicer.RoleCode)
		assert.Equal(t, "524802931", stlm.Invoicer.LegalOrganization.ID.Value)

		// EXT-FR-FE-BG-04, pinned to UNCL 3035 "IV" by EXT-FR-FE-90.
		require.NotNil(t, stlm.Invoicee)
		assert.Equal(t, "Adressée SAS", stlm.Invoicee.Name)
		assert.Equal(t, "IV", stlm.Invoicee.RoleCode)
		require.NotEmpty(t, stlm.Invoicee.GlobalID)
		assert.Equal(t, "31419443800017", stlm.Invoicee.GlobalID[0].Value)
		assert.Equal(t, "0009", stlm.Invoicee.GlobalID[0].SchemeID)

		// EXT-FR-FE-BG-02.
		require.NotNil(t, stlm.Payer)
		assert.Equal(t, "Payeur SA", stlm.Payer.Name)
		require.NotEmpty(t, stlm.Payer.SpecifiedTaxRegistration)
		assert.Equal(t, "FR44391838042", stlm.Payer.SpecifiedTaxRegistration[0].ID.Value)
	})

	t.Run("french extended maps the seller and buyer agents", func(t *testing.T) {
		agmt := convert(t, cii.ContextPeppolFranceExtendedV1).Transaction.Agreement

		// GOBL nests the agents in the party they act for; CII keeps them as
		// siblings in the trade agreement.
		require.NotNil(t, agmt.SalesAgent)
		assert.Equal(t, "Agent de Vendeur SAS", agmt.SalesAgent.Name)
		assert.Equal(t, "443061841", agmt.SalesAgent.LegalOrganization.ID.Value)

		require.NotNil(t, agmt.BuyerAgent)
		assert.Equal(t, "Agence Media SARL", agmt.BuyerAgent.Name)
		assert.Equal(t, "FR96552100554", agmt.BuyerAgent.SpecifiedTaxRegistration[0].ID.Value)
	})

	t.Run("the factur-x flavour of the profile maps them too", func(t *testing.T) {
		// ContextPeppolFranceFacturXV1 declares the Factur-X EXTENDED
		// guideline in BT-24 but is checked by the same EXTENDED-CTC-FR
		// rule set, so it carries the same parties.
		tr := convert(t, cii.ContextPeppolFranceFacturXV1).Transaction

		require.NotNil(t, tr.Settlement.Invoicee)
		require.NotNil(t, tr.Settlement.Payer)
		require.NotNil(t, tr.Agreement.SalesAgent)
		require.NotNil(t, tr.Agreement.BuyerAgent)
		assert.Equal(t, "II", tr.Settlement.Invoicer.RoleCode)
	})

	t.Run("extended-only parties are ignored outside the french extended contexts", func(t *testing.T) {
		// The CIUS profile is the closest neighbour that must not carry them.
		for _, ctx := range []cii.Context{cii.ContextEN16931V2017, cii.ContextPeppolFranceCIUSV1} {
			tr := convert(t, ctx).Transaction
			assert.Nil(t, tr.Settlement.Invoicee)
			assert.Nil(t, tr.Settlement.Payer)
			assert.Nil(t, tr.Agreement.SalesAgent)
			assert.Nil(t, tr.Agreement.BuyerAgent)
			require.NotNil(t, tr.Settlement.Invoicer)
			assert.Empty(t, tr.Settlement.Invoicer.RoleCode)
		}
	})

	t.Run("parse restores every extended party", func(t *testing.T) {
		data, err := convert(t, cii.ContextPeppolFranceExtendedV1).Bytes()
		require.NoError(t, err)

		env, err := cii.Parse(data)
		require.NoError(t, err)
		out, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, out.Ordering)
		require.NotNil(t, out.Ordering.Issuer)
		assert.Equal(t, "Facturant SARL", out.Ordering.Issuer.Name)
		require.NotNil(t, out.Ordering.Buyer)
		assert.Equal(t, "Adressée SAS", out.Ordering.Buyer.Name)
		require.NotNil(t, out.Payment)
		require.NotNil(t, out.Payment.Payer)
		assert.Equal(t, "Payeur SA", out.Payment.Payer.Name)
		require.NotNil(t, out.Supplier.Agent)
		assert.Equal(t, "Agent de Vendeur SAS", out.Supplier.Agent.Name)
		assert.Nil(t, out.Supplier.Agent.Agent)
		require.NotNil(t, out.Customer.Agent)
		assert.Equal(t, "Agence Media SARL", out.Customer.Agent.Name)
	})
}
