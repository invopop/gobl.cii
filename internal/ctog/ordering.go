package ctog

import (
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

// Values for AdditionalReferencedDocument.TypeCode
const (
	AdditionalDocumentTypeTender         = "50"
	AdditionalDocumentTypeProductInvoice = "130"
)

const (
	keyAdditionalDocumentTypeInvoiceDataSheet = "invoice-data-sheet"
	keyAdditionalDocumentTypeRefPaper         = "ref-paper"
)

func (c *Converter) prepareOrdering(doc *document.Document) error {
	ord := &bill.Ordering{}

	if doc.Transaction.Agreement.BuyerReference != "" {
		ord.Code = cbc.Code(doc.Transaction.Agreement.BuyerReference)
	}

	// Ordering period parsing
	if doc.Transaction.Settlement.Period != nil {
		per := &cal.Period{}

		if doc.Transaction.Settlement.Period.Start != nil && doc.Transaction.Settlement.Period.Start.DateFormat != nil {
			start, err := ParseDate(doc.Transaction.Settlement.Period.Start.DateFormat.Value)
			if err != nil {
				return err
			}
			per.Start = start
		}

		if doc.Transaction.Settlement.Period.End != nil && doc.Transaction.Settlement.Period.End.DateFormat != nil {
			end, err := ParseDate(doc.Transaction.Settlement.Period.End.DateFormat.Value)
			if err != nil {
				return err
			}
			per.End = end
		}
		if doc.Transaction.Settlement.Period.Description != nil {
			per.Label = *doc.Transaction.Settlement.Period.Description
		}
		ord.Period = per
	}

	// Despatch, Receiving and Tender parsing
	if doc.Transaction.Delivery.Despatch != nil {
		ord.Despatch = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.Transaction.Delivery.Despatch.ID),
			},
		}
	}

	if doc.Transaction.Delivery.Receiving != nil {
		ord.Receiving = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.Transaction.Delivery.Receiving.ID),
			},
		}
	}

	if len(doc.Transaction.Agreement.AdditionalDocument) > 0 {
		for _, ref := range doc.Transaction.Agreement.AdditionalDocument {
			switch ref.TypeCode {
			case AdditionalDocumentTypeTender:
				if ord.Tender == nil {
					ord.Tender = make([]*org.DocumentRef, 0)
				}
				docRef := &org.DocumentRef{
					Code: cbc.Code(ref.ID),
				}
				if ref.IssueDate != nil && ref.IssueDate.DateFormat != nil {
					refDate, err := ParseDate(ref.IssueDate.DateFormat.Value)
					if err != nil {
						return err
					}
					docRef.IssueDate = &refDate
				}
				ord.Tender = append(ord.Tender, docRef)
			case AdditionalDocumentTypeProductInvoice:
				if ord.Identities == nil {
					ord.Identities = make([]*org.Identity, 0)
				}
				ord.Identities = append(ord.Identities, &org.Identity{
					Key:  keyAdditionalDocumentTypeInvoiceDataSheet,
					Code: cbc.Code(ref.ID),
				})
			default:
				ord.Identities = append(ord.Identities, &org.Identity{
					Key:  keyAdditionalDocumentTypeRefPaper,
					Code: cbc.Code(ref.ID),
				})
			}
		}
	}

	if ord.Code != "" || ord.Period != nil || ord.Despatch != nil || ord.Receiving != nil || ord.Tender != nil || ord.Identities != nil {
		c.inv.Ordering = ord
	}
	return nil
}
