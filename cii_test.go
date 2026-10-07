package cii_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// example1 reads the CII_example1.xml parse fixture.
func example1(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(getParsePath(), "CII_example1.xml"))
	require.NoError(t, err)
	return data
}

func TestExport(t *testing.T) {
	t.Run("with default format", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		out, err := cii.ExportInvoice(env)
		require.NoError(t, err)

		assert.Equal(t, "urn:cen.eu:en16931:2017", out.ExchangedContext.GuidelineContext.ID)
		assert.Nil(t, out.ExchangedContext.BusinessContext)
	})

	t.Run("with missing addon", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		inv := env.Extract().(*bill.Invoice)
		inv.SetAddons() // empty
		_, err := cii.ExportInvoice(env)
		assert.ErrorContains(t, err, "gobl invoice missing addon eu-en16931-v2017")
	})

	t.Run("with Peppol format", func(t *testing.T) {
		env := loadEnvelope(t, "peppol/invoice-complete.json")
		out, err := cii.ExportInvoice(env, cii.WithFormat(cii.FormatPeppol))
		require.NoError(t, err)

		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", out.ExchangedContext.BusinessContext.ID)
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", out.ExchangedContext.GuidelineContext.ID)
	})

	t.Run("with output guideline", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		f := cii.FormatEN16931
		f.OutputGuidelineID = "urn:test:output"
		out, err := cii.ExportInvoice(env, cii.WithFormat(f))
		require.NoError(t, err)
		assert.Equal(t, "urn:test:output", out.ExchangedContext.GuidelineContext.ID)
	})

	t.Run("not an invoice", func(t *testing.T) {
		env := gobl.NewEnvelope()
		require.NoError(t, env.Insert(&bill.Status{}))
		_, err := cii.Export(env)
		assert.ErrorIs(t, err, cii.ErrUnsupportedDocumentType)
		_, err = cii.ExportInvoice(env)
		assert.ErrorIs(t, err, cii.ErrUnsupportedDocumentType)
	})

	t.Run("export functions run in order", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		var calls []string
		f := cii.FormatEN16931
		f.ExportFuncs = []cii.ExportFunc{
			func(f *cii.Format, _ *gobl.Envelope, doc cii.Document) error {
				calls = append(calls, "first")
				doc.(*cii.Invoice).ExchangedContext.GuidelineContext.ID = f.Key.String()
				return nil
			},
			func(_ *cii.Format, _ *gobl.Envelope, _ cii.Document) error {
				calls = append(calls, "second")
				return nil
			},
		}
		out, err := cii.ExportInvoice(env, cii.WithFormat(f))
		require.NoError(t, err)
		assert.Equal(t, []string{"first", "second"}, calls)
		assert.Equal(t, "cii+en16931", out.ExchangedContext.GuidelineContext.ID)
	})

	t.Run("export function error", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		f := cii.FormatEN16931
		f.ExportFuncs = []cii.ExportFunc{
			func(_ *cii.Format, _ *gobl.Envelope, _ cii.Document) error {
				return errors.New("export failed")
			},
		}
		_, err := cii.Export(env, cii.WithFormat(f))
		assert.EqualError(t, err, "export failed")
	})
}

func TestImport(t *testing.T) {
	t.Run("detects the format", func(t *testing.T) {
		env, err := parseCII(example1(t))
		require.NoError(t, err)
		inv := env.Extract().(*bill.Invoice)
		assert.Contains(t, inv.GetAddons(), cii.FormatEN16931.Addons[0])
	})

	t.Run("with format", func(t *testing.T) {
		f := cii.FormatEN16931
		f.Addons = nil
		env, err := parseCII(example1(t), cii.WithFormat(f))
		require.NoError(t, err)
		inv := env.Extract().(*bill.Invoice)
		assert.Empty(t, inv.GetAddons())
	})

	t.Run("import functions run in order", func(t *testing.T) {
		var calls []string
		f := cii.FormatEN16931
		f.ImportFuncs = []cii.ImportFunc{
			func(_ *cii.Format, doc cii.Document, env *gobl.Envelope) error {
				calls = append(calls, "first")
				env.Extract().(*bill.Invoice).Code = cbc.Code(doc.(*cii.Invoice).ExchangedDocument.ID + "-X")
				return nil
			},
			func(_ *cii.Format, _ cii.Document, _ *gobl.Envelope) error {
				calls = append(calls, "second")
				return nil
			},
		}
		env, err := parseCII(example1(t), cii.WithFormat(f))
		require.NoError(t, err)
		assert.Equal(t, []string{"first", "second"}, calls)
		assert.Equal(t, "12115118-X", env.Extract().(*bill.Invoice).Code.String())
	})

	t.Run("import function error", func(t *testing.T) {
		f := cii.FormatEN16931
		f.ImportFuncs = []cii.ImportFunc{
			func(_ *cii.Format, _ cii.Document, _ *gobl.Envelope) error {
				return errors.New("import failed")
			},
		}
		_, err := parseCII(example1(t), cii.WithFormat(f))
		assert.EqualError(t, err, "import failed")
	})

	t.Run("calculation error", func(t *testing.T) {
		f := cii.FormatEN16931
		f.ImportFuncs = []cii.ImportFunc{
			func(_ *cii.Format, _ cii.Document, env *gobl.Envelope) error {
				env.Document = nil
				return nil
			},
		}
		_, err := parseCII(example1(t), cii.WithFormat(f))
		assert.Error(t, err)
	})

	t.Run("invalid issue date", func(t *testing.T) {
		doc, err := cii.Decode(example1(t))
		require.NoError(t, err)
		doc.(*cii.Invoice).ExchangedDocument.IssueDate.DateFormat.Value = "bad"
		_, err = cii.Import(doc)
		assert.Error(t, err)
	})

	t.Run("not an invoice", func(t *testing.T) {
		_, err := cii.Import(cii.NewCDAR())
		assert.ErrorIs(t, err, cii.ErrUnsupportedDocumentType)
	})
}

func TestDecodeEncode(t *testing.T) {
	t.Run("invoice", func(t *testing.T) {
		doc, err := cii.Decode(example1(t))
		require.NoError(t, err)
		require.IsType(t, &cii.Invoice{}, doc)
		data, err := cii.Encode(doc)
		require.NoError(t, err)
		assert.Contains(t, string(data), "<?xml")
	})

	t.Run("CDAR", func(t *testing.T) {
		in := cii.NewCDAR()
		in.ExchangedDocument = &cii.CDARExchangedDocument{ID: "CDAR-1"}
		data, err := cii.Encode(in)
		require.NoError(t, err)
		assert.Contains(t, string(data), cii.NamespaceCDARRSM)

		doc, err := cii.Decode(data)
		require.NoError(t, err)
		out, ok := doc.(*cii.CDAR)
		require.True(t, ok)
		assert.Equal(t, "CDAR-1", out.ExchangedDocument.ID)
	})

	t.Run("malformed CDAR", func(t *testing.T) {
		data := []byte(`<rsm:CrossDomainAcknowledgementAndResponse xmlns:rsm="` + cii.NamespaceCDARRSM + `"><rsm:ExchangedDocument>`)
		_, err := cii.Decode(data)
		assert.ErrorContains(t, err, "error unmarshaling CDAR")
	})

	t.Run("unknown namespace", func(t *testing.T) {
		_, err := cii.Decode([]byte(`<Invoice xmlns="urn:test"/>`))
		assert.ErrorIs(t, err, cii.ErrUnknownDocumentType)
	})

	t.Run("no root element", func(t *testing.T) {
		_, err := cii.Decode([]byte(`<?xml version="1.0"?>`))
		assert.ErrorIs(t, err, cii.ErrUnknownDocumentType)
	})

	t.Run("invalid XML", func(t *testing.T) {
		_, err := cii.Decode([]byte(`<<`))
		assert.ErrorContains(t, err, "error parsing XML")
	})

	t.Run("no document", func(t *testing.T) {
		_, err := cii.Encode(nil)
		assert.ErrorIs(t, err, cii.ErrUnsupportedDocumentType)
	})
}

func TestCleanString(t *testing.T) {
	assert.Equal(t, "Cafe", cii.CleanString("Caf�e"))
}
