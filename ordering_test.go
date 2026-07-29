package cii_test

import (
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
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
