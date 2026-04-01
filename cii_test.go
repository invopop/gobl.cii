package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertInvoiceWithContext(t *testing.T) {
	env := loadEnvelope(t, "en16931/invoice-complete.json")

	t.Run("with default context", func(t *testing.T) {
		out, err := cii.ConvertInvoice(env)
		require.NoError(t, err)

		assert.Equal(t, "urn:cen.eu:en16931:2017", out.ExchangedContext.GuidelineContext.ID)
		assert.Nil(t, out.ExchangedContext.BusinessContext)
	})

	t.Run("with missing addon", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		inv := env.Extract().(*bill.Invoice)
		inv.SetAddons() // empty
		_, err := cii.ConvertInvoice(env)
		assert.ErrorContains(t, err, "gobl invoice missing addon eu-en16931-v2017")
	})

	t.Run("with Factur-X context", func(t *testing.T) {
		env := loadEnvelope(t, "facturx/invoice-complete.json")
		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextFacturXV1))
		require.NoError(t, err)

		assert.Equal(t, "urn:cen.eu:en16931:2017", out.ExchangedContext.GuidelineContext.ID)
		assert.Nil(t, out.ExchangedContext.BusinessContext)
	})

	t.Run("with XRechnung context", func(t *testing.T) {
		env := loadEnvelope(t, "xrechnung/invoice-de-de.json")
		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextXRechnungV3))
		require.NoError(t, err)

		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0", out.ExchangedContext.GuidelineContext.ID)
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", out.ExchangedContext.BusinessContext.ID)
	})

	t.Run("with PEPPOL context", func(t *testing.T) {
		env := loadEnvelope(t, "peppol/invoice-complete.json")
		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextPeppolV3))
		require.NoError(t, err)

		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", out.ExchangedContext.BusinessContext.ID)
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", out.ExchangedContext.GuidelineContext.ID)
	})
}

func TestFindContext(t *testing.T) {
	t.Run("find EN16931 by GuidelineID", func(t *testing.T) {
		ctx := cii.FindContext("urn:cen.eu:en16931:2017", "")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextEN16931V2017.GuidelineID, ctx.GuidelineID)
	})

	t.Run("find EN16931 with non-billing-mode BusinessID", func(t *testing.T) {
		// EN16931 documents may have arbitrary BusinessIDs that are not French billing modes
		ctx := cii.FindContext("urn:cen.eu:en16931:2017", "some-business-process")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextEN16931V2017.GuidelineID, ctx.GuidelineID)
	})

	t.Run("find Peppol by GuidelineID and BusinessID", func(t *testing.T) {
		ctx := cii.FindContext("urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextPeppolV3.GuidelineID, ctx.GuidelineID)
	})

	t.Run("find France CIUS by full GuidelineID", func(t *testing.T) {
		ctx := cii.FindContext("urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0", "urn:peppol:france:billing:regulated")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextPeppolFranceCIUSV1.GuidelineID, ctx.GuidelineID)
	})

	t.Run("find France CIUS by billing mode BusinessID", func(t *testing.T) {
		// France CIUS documents use EN16931 GuidelineID but have a billing mode as BusinessID
		ctx := cii.FindContext("urn:cen.eu:en16931:2017", "B1")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextPeppolFranceCIUSV1.GuidelineID, ctx.GuidelineID)

		ctx = cii.FindContext("urn:cen.eu:en16931:2017", "S1")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextPeppolFranceCIUSV1.GuidelineID, ctx.GuidelineID)

		ctx = cii.FindContext("urn:cen.eu:en16931:2017", "M4")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextPeppolFranceCIUSV1.GuidelineID, ctx.GuidelineID)
	})

	t.Run("find France Factur-X by OutputGuidelineID", func(t *testing.T) {
		ctx := cii.FindContext("urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr", "")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextPeppolFranceFacturXV1.GuidelineID, ctx.GuidelineID)
	})

	t.Run("find France Factur-X by billing mode BusinessID", func(t *testing.T) {
		ctx := cii.FindContext("urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr", "B1")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextPeppolFranceFacturXV1.GuidelineID, ctx.GuidelineID)
	})

	t.Run("find XRechnung by GuidelineID", func(t *testing.T) {
		ctx := cii.FindContext("urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0", "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0")
		require.NotNil(t, ctx)
		assert.Equal(t, cii.ContextXRechnungV3.GuidelineID, ctx.GuidelineID)
	})

	t.Run("unknown GuidelineID returns nil", func(t *testing.T) {
		ctx := cii.FindContext("unknown:guideline:id", "")
		assert.Nil(t, ctx)
	})
}
