package cii

import (
	"encoding/base64"
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// emptyInvoice builds the minimum structure the attachment helpers write into.
func emptyInvoice() *Invoice {
	return &Invoice{
		Transaction: &Transaction{
			Agreement: &Agreement{},
		},
	}
}

func TestAddAttachments(t *testing.T) {
	t.Run("every field maps across", func(t *testing.T) {
		out := emptyInvoice()
		out.addAttachments(&bill.Invoice{Attachments: []*org.Attachment{
			{Code: testDocID, Description: "Timesheet", URL: "https://example.com/ts.pdf"},
		}})

		docs := out.Transaction.Agreement.AdditionalDocument
		require.Len(t, docs, 1)
		assert.Equal(t, AdditionalDocumentTypeAttachment, docs[0].TypeCode)
		assert.Equal(t, testDocID, docs[0].ID)
		assert.Equal(t, "Timesheet", docs[0].Name)
		assert.Equal(t, "https://example.com/ts.pdf", docs[0].URIID)
	})

	t.Run("an attachment with nothing set still declares its type", func(t *testing.T) {
		out := emptyInvoice()
		out.addAttachments(&bill.Invoice{Attachments: []*org.Attachment{{}}})

		docs := out.Transaction.Agreement.AdditionalDocument
		require.Len(t, docs, 1)
		assert.Equal(t, AdditionalDocumentTypeAttachment, docs[0].TypeCode)
		assert.Empty(t, docs[0].ID)
	})

	t.Run("no attachments", func(t *testing.T) {
		out := emptyInvoice()
		out.addAttachments(&bill.Invoice{})
		assert.Empty(t, out.Transaction.Agreement.AdditionalDocument)
	})

	t.Run("several are kept in order", func(t *testing.T) {
		out := emptyInvoice()
		out.addAttachments(&bill.Invoice{Attachments: []*org.Attachment{
			{Code: "A"}, {Code: "B"},
		}})
		docs := out.Transaction.Agreement.AdditionalDocument
		require.Len(t, docs, 2)
		assert.Equal(t, "A", docs[0].ID)
		assert.Equal(t, "B", docs[1].ID)
	})
}

func TestGoblAttachments(t *testing.T) {
	t.Run("every field maps back", func(t *testing.T) {
		got := goblAttachments([]*AdditionalDocument{{
			TypeCode: AdditionalDocumentTypeAttachment,
			ID:       testDocID,
			Name:     "Timesheet",
			URIID:    "https://example.com/ts.pdf",
		}})
		require.Len(t, got, 1)
		assert.Equal(t, testDocID, got[0].Code.String())
		assert.Equal(t, "Timesheet", got[0].Description)
		assert.Equal(t, "https://example.com/ts.pdf", got[0].URL)
	})

	t.Run("documents of another type are not attachments", func(t *testing.T) {
		// 130 is an invoice data sheet, 50 a price list: neither is BG-24.
		assert.Empty(t, goblAttachments([]*AdditionalDocument{
			{TypeCode: "130", ID: "X"},
			{TypeCode: "50", ID: "Y"},
			{ID: "no type code"},
		}))
	})

	t.Run("embedded binaries are left to ExtractBinaryAttachments", func(t *testing.T) {
		assert.Empty(t, goblAttachments([]*AdditionalDocument{{
			TypeCode:               AdditionalDocumentTypeAttachment,
			ID:                     testDocID,
			AttachmentBinaryObject: &BinaryObject{Value: "AAAA"},
		}}))
	})

	t.Run("nothing to convert", func(t *testing.T) {
		assert.Empty(t, goblAttachments(nil))
	})
}

func TestBinaryAttachments(t *testing.T) {
	payload := []byte("%PDF-1.4 not really a pdf")

	t.Run("round trips through base64", func(t *testing.T) {
		out := emptyInvoice()
		out.AddBinaryAttachment(BinaryAttachment{
			ID:          testDocID,
			Description: "Invoice PDF",
			Data:        payload,
			MimeCode:    "application/pdf",
			Filename:    "invoice.pdf",
		})

		docs := out.Transaction.Agreement.AdditionalDocument
		require.Len(t, docs, 1)
		require.NotNil(t, docs[0].AttachmentBinaryObject)
		assert.Equal(t, AdditionalDocumentTypeAttachment, docs[0].TypeCode)
		assert.Equal(t, base64.StdEncoding.EncodeToString(payload), docs[0].AttachmentBinaryObject.Value)

		got := out.ExtractBinaryAttachments()
		require.Len(t, got, 1)
		assert.Equal(t, testDocID, got[0].ID)
		assert.Equal(t, "Invoice PDF", got[0].Description)
		assert.Equal(t, payload, got[0].Data)
		assert.Equal(t, "application/pdf", got[0].MimeCode)
		assert.Equal(t, "invoice.pdf", got[0].Filename)
	})

	t.Run("whitespace from XML formatting is ignored", func(t *testing.T) {
		// An indented document wraps the base64 across lines, which is not
		// valid base64 on its own.
		encoded := base64.StdEncoding.EncodeToString(payload)
		out := emptyInvoice()
		out.Transaction.Agreement.AdditionalDocument = []*AdditionalDocument{{
			TypeCode: AdditionalDocumentTypeAttachment,
			AttachmentBinaryObject: &BinaryObject{
				Value: "\n        " + encoded[:8] + "\n        " + encoded[8:] + "\n      ",
			},
		}}

		got := out.ExtractBinaryAttachments()
		require.Len(t, got, 1)
		assert.Equal(t, payload, got[0].Data)
	})

	t.Run("undecodable data is skipped rather than fatal", func(t *testing.T) {
		out := emptyInvoice()
		out.Transaction.Agreement.AdditionalDocument = []*AdditionalDocument{
			{TypeCode: AdditionalDocumentTypeAttachment, AttachmentBinaryObject: &BinaryObject{Value: "!!not base64!!"}},
			{TypeCode: AdditionalDocumentTypeAttachment, AttachmentBinaryObject: &BinaryObject{Value: base64.StdEncoding.EncodeToString(payload)}},
		}

		got := out.ExtractBinaryAttachments()
		require.Len(t, got, 1, "the readable attachment should still come back")
		assert.Equal(t, payload, got[0].Data)
	})

	t.Run("documents without binary data are skipped", func(t *testing.T) {
		out := emptyInvoice()
		out.Transaction.Agreement.AdditionalDocument = []*AdditionalDocument{
			{TypeCode: AdditionalDocumentTypeAttachment, ID: "no binary"},
			{TypeCode: AdditionalDocumentTypeAttachment, AttachmentBinaryObject: &BinaryObject{Value: ""}},
		}
		assert.Empty(t, out.ExtractBinaryAttachments())
	})

	t.Run("no attachments at all", func(t *testing.T) {
		assert.Empty(t, emptyInvoice().ExtractBinaryAttachments())
	})

	t.Run("empty data still round trips", func(t *testing.T) {
		out := emptyInvoice()
		out.AddBinaryAttachment(BinaryAttachment{ID: "EMPTY"})
		// An empty payload encodes to an empty string, which reads back as
		// nothing rather than as a zero-length attachment.
		assert.Empty(t, out.ExtractBinaryAttachments())
	})
}
