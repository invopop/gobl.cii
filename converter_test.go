package cii_test

import (
	"strings"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverter(t *testing.T) {
	t.Run("formats", func(t *testing.T) {
		for _, key := range []cbc.Key{cii.KeyCII, cii.FormatEN16931.Key, cii.FormatPeppol.Key} {
			f := convert.FormatFor(key)
			require.NotNil(t, f, key)
			assert.Equal(t, "application/xml", f.MIME)
		}
		f := convert.FormatFor(cii.KeyCII)
		assert.NotEmpty(t, f.Import)
		assert.Empty(t, f.Export, "unknown specifications are not exported")
	})

	t.Run("detect", func(t *testing.T) {
		f, err := convert.Detect(example1(t))
		require.NoError(t, err)
		assert.Equal(t, cii.FormatEN16931.Key, f.Key)
	})

	t.Run("detect a registered format", func(t *testing.T) {
		data := strings.Replace(string(example1(t)), cii.GuidelineIDEN16931, "urn:test:output-out", 1)
		f, err := convert.Detect([]byte(data))
		require.NoError(t, err)
		assert.Equal(t, formatTestOutput.Key, f.Key)
	})

	t.Run("detect an unknown specification", func(t *testing.T) {
		data := strings.Replace(string(example1(t)), cii.GuidelineIDEN16931, "urn:test:unknown", 1)
		f, err := convert.Detect([]byte(data))
		require.NoError(t, err)
		assert.Equal(t, cii.KeyCII, f.Key)

		env, err := convert.Import([]byte(data))
		require.NoError(t, err)
		_, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)
	})

	t.Run("ignore other documents", func(t *testing.T) {
		in := convert.NewInput([]byte(`<rsm:CrossDomainAcknowledgementAndResponse xmlns:rsm="` + cii.NamespaceCDARRSM + `"/>`))
		assert.Equal(t, cii.NamespaceCDARRSM, cii.ReadHeader(in).Namespace)
		_, err := convert.Detect(in.Data, cii.KeyCII)
		assert.Error(t, err)
		_, err = convert.Detect([]byte("not xml"), cii.KeyCII)
		assert.Error(t, err)
	})

	t.Run("import", func(t *testing.T) {
		env, err := convert.Import(example1(t))
		require.NoError(t, err)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.Equal(t, "12115118", inv.Code.String())

		_, err = convert.Import([]byte(`<rsm:CrossIndustryInvoice xmlns:rsm="`+cii.NamespaceRSM+`"><`), cii.KeyCII)
		assert.Error(t, err)
	})

	t.Run("export and detect again", func(t *testing.T) {
		env := loadEnvelope(t, "peppol/invoice-complete.json")
		out, err := convert.Export(env, cii.FormatPeppol.Key)
		require.NoError(t, err)
		assert.Equal(t, cii.FormatPeppol.Key, out.Format.Key)

		f, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, cii.FormatPeppol.Key, f.Key)
	})

	t.Run("export error", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		env.Extract().(*bill.Invoice).SetAddons()
		_, err := convert.Export(env, cii.FormatEN16931.Key)
		assert.Error(t, err)
	})
}

func TestReadHeader(t *testing.T) {
	t.Run("invoice", func(t *testing.T) {
		in := convert.NewInput(example1(t))
		h := cii.ReadHeader(in)
		require.NoError(t, h.Err)
		assert.Equal(t, cii.NamespaceRSM, h.Namespace)
		assert.Equal(t, cii.GuidelineIDEN16931, h.GuidelineID)
		assert.Same(t, h, cii.ReadHeader(in), "read once")
	})

	t.Run("business process", func(t *testing.T) {
		env := loadEnvelope(t, "peppol/invoice-complete.json")
		out, err := convert.Export(env, cii.FormatPeppol.Key)
		require.NoError(t, err)
		h := cii.ReadHeader(convert.NewInput(out.Data))
		assert.Equal(t, cii.ProfileIDPeppolBilling, h.BusinessID)
	})

	t.Run("CDAR", func(t *testing.T) {
		data := `<rsm:CrossDomainAcknowledgementAndResponse xmlns:rsm="` + cii.NamespaceCDARRSM + `" xmlns:ram="` + cii.NamespaceRAM + `">` +
			`<rsm:ExchangedDocumentContext><ram:BusinessProcessSpecifiedDocumentContextParameter><ram:ID>REGULATED</ram:ID></ram:BusinessProcessSpecifiedDocumentContextParameter>` +
			`<ram:GuidelineSpecifiedDocumentContextParameter><ram:ID>urn:test:cdar</ram:ID></ram:GuidelineSpecifiedDocumentContextParameter></rsm:ExchangedDocumentContext>` +
			`</rsm:CrossDomainAcknowledgementAndResponse>`
		h := cii.ReadHeader(convert.NewInput([]byte(data)))
		require.NoError(t, h.Err)
		assert.Equal(t, cii.NamespaceCDARRSM, h.Namespace)
		assert.Equal(t, "urn:test:cdar", h.GuidelineID)
		assert.Equal(t, "REGULATED", h.BusinessID)
	})

	t.Run("no context", func(t *testing.T) {
		h := cii.ReadHeader(convert.NewInput([]byte(`<a:Root xmlns:a="urn:test"><a:Other/></a:Root>`)))
		require.NoError(t, h.Err)
		assert.Equal(t, "urn:test", h.Namespace)
		assert.Empty(t, h.GuidelineID)
	})

	t.Run("malformed context", func(t *testing.T) {
		h := cii.ReadHeader(convert.NewInput([]byte(`<a:Root xmlns:a="urn:test"><a:ExchangedDocumentContext><a:GuidelineSpecifiedDocumentContextParameter></a:Root>`)))
		assert.Error(t, h.Err)
	})

	t.Run("empty", func(t *testing.T) {
		h := cii.ReadHeader(convert.NewInput(nil))
		assert.ErrorIs(t, h.Err, cii.ErrUnknownDocumentType)
	})
}
