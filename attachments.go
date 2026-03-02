package cii

import (
	"encoding/base64"
	"regexp"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

// AdditionalDocumentTypeAttachment is the TypeCode for additional supporting documents
const AdditionalDocumentTypeAttachment = "916"

// BinaryAttachment represents a binary attachment that can be extracted from
// or added to a CII invoice.
type BinaryAttachment struct {
	// ID is the identifier for this attachment reference
	ID string
	// Description provides a human-readable description of the attachment
	Description string
	// Data contains the raw binary data (automatically base64-encoded/decoded as needed)
	Data []byte
	// MimeCode specifies the MIME type (e.g., "application/pdf")
	MimeCode string
	// Filename is the name of the file
	Filename string
}

// addAttachments converts GOBL attachments to CII AdditionalReferencedDocuments
func (out *Invoice) addAttachments(inv *bill.Invoice) {
	for _, a := range inv.Attachments {
		doc := &AdditionalDocument{
			TypeCode: AdditionalDocumentTypeAttachment,
		}
		if a.Code != "" {
			doc.ID = a.Code.String()
		}
		if a.Description != "" {
			doc.Name = a.Description
		}
		if a.URL != "" {
			doc.URIID = a.URL
		}
		out.Transaction.Agreement.AdditionalDocument = append(
			out.Transaction.Agreement.AdditionalDocument,
			doc,
		)
	}
}

// goblAttachments converts CII AdditionalReferencedDocuments to GOBL attachments.
func goblAttachments(docs []*AdditionalDocument) []*org.Attachment {
	var attachments []*org.Attachment
	for _, doc := range docs {
		if doc.TypeCode != AdditionalDocumentTypeAttachment {
			continue
		}
		// Skip binary attachments — handled by ExtractBinaryAttachments
		if doc.AttachmentBinaryObject != nil {
			continue
		}
		att := &org.Attachment{}
		if doc.ID != "" {
			att.Code = cbc.Code(doc.ID)
		}
		if doc.Name != "" {
			att.Description = doc.Name
		}
		if doc.URIID != "" {
			att.URL = doc.URIID
		}
		attachments = append(attachments, att)
	}
	return attachments
}

// AddBinaryAttachment adds an embedded binary attachment to the CII Invoice.
// The binary data will be automatically base64-encoded.
func (out *Invoice) AddBinaryAttachment(att BinaryAttachment) {
	encodedData := base64.StdEncoding.EncodeToString(att.Data)

	doc := &AdditionalDocument{
		TypeCode: AdditionalDocumentTypeAttachment,
		ID:       att.ID,
		Name:     att.Description,
		AttachmentBinaryObject: &BinaryObject{
			Value:    encodedData,
			MimeCode: att.MimeCode,
			Filename: att.Filename,
		},
	}

	out.Transaction.Agreement.AdditionalDocument = append(
		out.Transaction.Agreement.AdditionalDocument,
		doc,
	)
}

// ExtractBinaryAttachments extracts all binary attachments from the CII Invoice.
// It returns a slice of BinaryAttachment containing the ID, description, and decoded binary data.
func (out *Invoice) ExtractBinaryAttachments() []BinaryAttachment {
	var result []BinaryAttachment

	for _, doc := range out.Transaction.Agreement.AdditionalDocument {
		if doc.AttachmentBinaryObject == nil || doc.AttachmentBinaryObject.Value == "" {
			continue
		}

		// Remove whitespace that may have been added by XML formatting
		whitespaceRegex := regexp.MustCompile(`\s+`)
		v := whitespaceRegex.ReplaceAllString(doc.AttachmentBinaryObject.Value, "")

		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			continue
		}

		result = append(result, BinaryAttachment{
			ID:          doc.ID,
			Description: doc.Name,
			Data:        decoded,
			MimeCode:    doc.AttachmentBinaryObject.MimeCode,
			Filename:    doc.AttachmentBinaryObject.Filename,
		})
	}

	return result
}
