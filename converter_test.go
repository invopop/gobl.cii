package cii_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/rules"
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

	t.Run("import an on-request format only when asked for", func(t *testing.T) {
		f, err := convert.Detect(example1(t))
		require.NoError(t, err)
		assert.Equal(t, cii.FormatEN16931.Key, f.Key, "never detected on its own")

		res, err := convert.Import(example1(t), convert.WithFormats(formatTestOnRequest.Key))
		require.NoError(t, err)
		assert.Equal(t, formatTestOnRequest.Key, res.Format.Key)
	})

	t.Run("a requested format wins over a match", func(t *testing.T) {
		data := strings.Replace(string(example1(t)), "<ram:GuidelineSpecifiedDocumentContextParameter>",
			"<ram:BusinessProcessSpecifiedDocumentContextParameter><ram:ID>test-match</ram:ID></ram:BusinessProcessSpecifiedDocumentContextParameter><ram:GuidelineSpecifiedDocumentContextParameter>", 1)
		f, err := convert.Detect([]byte(data))
		require.NoError(t, err)
		assert.Equal(t, formatTestMatch.Key, f.Key)

		res, err := convert.Import([]byte(data), convert.WithFormats(cii.FormatEN16931.Key))
		require.NoError(t, err)
		assert.Equal(t, cii.FormatEN16931.Key, res.Format.Key)
	})

	t.Run("a requested format the document can't be in is refused", func(t *testing.T) {
		_, err := convert.Import(example1(t), convert.WithFormats(formatTestOutput.Key))
		assert.ErrorIs(t, err, convert.ErrUnknownFormat)
	})

	t.Run("detect an unknown specification", func(t *testing.T) {
		data := strings.Replace(string(example1(t)), cii.GuidelineIDEN16931, "urn:test:unknown", 1)
		f, err := convert.Detect([]byte(data))
		require.NoError(t, err)
		assert.Equal(t, cii.KeyCII, f.Key)

		res, err := convert.Import([]byte(data))
		require.NoError(t, err)
		assert.Equal(t, cii.KeyCII, res.Format.Key)
		_, ok := res.Envelope.Extract().(*bill.Invoice)
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
		res, err := convert.Import(example1(t))
		require.NoError(t, err)
		inv, ok := res.Envelope.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.Equal(t, "12115118", inv.Code.String())
		assert.Empty(t, res.Attachments)

		_, err = convert.Import([]byte(`<rsm:CrossIndustryInvoice xmlns:rsm="`+cii.NamespaceRSM+`"><`), convert.WithFormats(cii.KeyCII))
		assert.Error(t, err)
	})

	t.Run("import attachments", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(getParsePath(), "CII_example2.xml"))
		require.NoError(t, err)
		res, err := convert.Import(data)
		require.NoError(t, err)
		require.Len(t, res.Attachments, 1)
		a := res.Attachments[0]
		assert.Equal(t, "Doc1", a.ID)
		assert.Equal(t, "Timesheet", a.Description)
		assert.Equal(t, "EHF.pdf", a.Name)
		assert.Equal(t, "application/pdf", a.MIME)
		// The example's file is itself base64 text, of a PDF.
		assert.True(t, bytes.HasPrefix(a.Data, []byte("JVBERi0x")), "decoded")
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

	t.Run("missing addon", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		env.Extract().(*bill.Invoice).SetAddons()
		_, err := convert.Export(env, cii.FormatEN16931.Key)
		assert.ErrorIs(t, err, convert.ErrNotSupported)
		var ge *gobl.Error
		require.ErrorAs(t, err, &ge)
		require.NotNil(t, ge.Faults())
		f := ge.Faults().First()
		assert.Equal(t, rules.Code("CII-EN16931-V2017-BILL-INVOICE-01"), f.Code())
		assert.Equal(t, []string{"$.$addons"}, f.Paths())
	})
}

func TestConverterExportAttachments(t *testing.T) {
	pdf := &convert.Attachment{
		ID:          "INV-PDF",
		Name:        "invoice.pdf",
		Description: "Invoice PDF",
		MIME:        "application/pdf",
		Data:        []byte("%PDF-1.7"),
	}
	t.Run("embedded", func(t *testing.T) {
		env := loadEnvelope(t, "peppol/invoice-complete.json")
		csv := &convert.Attachment{Name: "lines.csv", MIME: "text/csv", Data: []byte("a,b")}
		out, err := convert.Export(env, cii.FormatPeppol.Key, convert.WithAttachments(pdf, csv))
		require.NoError(t, err)

		doc, err := cii.Decode(out.Data)
		require.NoError(t, err)
		atts := doc.(*cii.Invoice).ExtractBinaryAttachments()
		require.Len(t, atts, 2)
		assert.Equal(t, "INV-PDF", atts[0].ID)
		assert.Equal(t, "Invoice PDF", atts[0].Description)
		assert.Equal(t, "application/pdf", atts[0].MimeCode)
		assert.Equal(t, "invoice.pdf", atts[0].Filename)
		assert.Equal(t, []byte("%PDF-1.7"), atts[0].Data)
		assert.Equal(t, "lines.csv", atts[1].ID, "named by the file without an ID")

		res, err := convert.Import(out.Data)
		require.NoError(t, err)
		require.Len(t, res.Attachments, 2)
		assert.Equal(t, pdf, res.Attachments[0], "imported again")
		assert.Equal(t, []byte("a,b"), res.Attachments[1].Data)
	})
	t.Run("refused by formats that do not embed them", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-complete.json")
		_, err := convert.Export(env, formatTestExportOnly.Key, convert.WithAttachments(pdf))
		assert.ErrorIs(t, err, convert.ErrNotSupported)
		assert.ErrorContains(t, err, "does not embed attachments")
	})
	t.Run("formats", func(t *testing.T) {
		assert.True(t, convert.FormatFor(cii.FormatEN16931.Key).Attachments)
		assert.True(t, convert.FormatFor(cii.FormatPeppol.Key).Attachments)
		assert.False(t, convert.FormatFor(formatTestExportOnly.Key).Attachments)
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
