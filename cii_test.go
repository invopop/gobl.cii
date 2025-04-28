package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/fr/facturx"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertInvoiceWithContext(t *testing.T) {
	env := loadEnvelope(t, "invoice-complete.json")

	t.Run("with default context", func(t *testing.T) {
		out, err := cii.ConvertInvoice(env)
		require.NoError(t, err)

		assert.Equal(t, "urn:cen.eu:en16931:2017", out.ExchangedContext.GuidelineContext.ID)
		assert.Nil(t, out.ExchangedContext.BusinessContext)
	})

	t.Run("with missing addon", func(t *testing.T) {
		env := loadEnvelope(t, "invoice-complete.json")
		inv := env.Extract().(*bill.Invoice)
		inv.SetAddons() // empty
		_, err := cii.ConvertInvoice(env)
		assert.ErrorContains(t, err, "gobl invoice missing addon eu-en16931-v2017")
	})

	t.Run("with Factur-X context", func(t *testing.T) {
		env := loadEnvelope(t, "invoice-complete.json")
		inv := env.Extract().(*bill.Invoice)
		inv.Addons.List = append(inv.Addons.List, facturx.V1)
		require.NoError(t, inv.Calculate())

		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextFacturXV1))
		require.NoError(t, err)

		assert.Equal(t, "urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended", out.ExchangedContext.GuidelineContext.ID)
		assert.Nil(t, out.ExchangedContext.BusinessContext)
	})

	t.Run("with XRechnung context", func(t *testing.T) {
		env := loadEnvelope(t, "invoice-complete.json")
		inv := env.Extract().(*bill.Invoice)
		inv.Addons.List = append(inv.Addons.List, xrechnung.V3)
		require.NoError(t, inv.Calculate())

		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextXRechnungV3))
		require.NoError(t, err)

		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0", out.ExchangedContext.GuidelineContext.ID)
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", out.ExchangedContext.BusinessContext.ID)
	})

}
