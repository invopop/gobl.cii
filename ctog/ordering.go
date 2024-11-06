package ctog

import (
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

func (c *Converter) prepareOrdering(doc *Document) error {
	ord := &bill.Ordering{}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerReference != nil {
		ord.Code = cbc.Code(*doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerReference)
	}

	// Ordering period parsing
	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod != nil {
		per := &cal.Period{}

		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.StartDateTime != nil {
			start, err := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.StartDateTime.DateTimeString)
			if err != nil {
				return err
			}
			per.Start = start
		}

		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.EndDateTime != nil {
			end, err := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.EndDateTime.DateTimeString)
			if err != nil {
				return err
			}
			per.End = end
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.Description != nil {
			per.Label = *doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.Description
		}
		ord.Period = per
	}

	// Despatch, Receiving and Tender parsing
	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument != nil {
		ord.Despatch = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.IssuerAssignedID),
			},
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.FormattedIssueDateTime != nil {
			refDate, err := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.FormattedIssueDateTime.DateTimeString)
			if err != nil {
				return err
			}
			ord.Despatch[0].IssueDate = &refDate
		}
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument != nil {
		ord.Receiving = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.IssuerAssignedID),
			},
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.FormattedIssueDateTime != nil {
			refDate, err := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.FormattedIssueDateTime.DateTimeString)
			if err != nil {
				return err
			}
			ord.Receiving[0].IssueDate = &refDate
		}
	}

	if len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.AdditionalReferencedDocument) > 0 {
		for _, ref := range doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.AdditionalReferencedDocument {
			switch ref.TypeCode {
			case AdditionalDocumentTypeTender:
				if ord.Tender == nil {
					ord.Tender = make([]*org.DocumentRef, 0)
				}
				docRef := &org.DocumentRef{
					Code: cbc.Code(ref.IssuerAssignedID),
				}
				if ref.FormattedIssueDateTime != nil {
					refDate, err := ParseDate(ref.FormattedIssueDateTime.DateTimeString)
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
					Code: cbc.Code(ref.IssuerAssignedID),
				})
			default:
				ord.Identities = append(ord.Identities, &org.Identity{
					Key:  keyAdditionalDocumentTypeRefPaper,
					Code: cbc.Code(ref.IssuerAssignedID),
				})
			}
		}
	}

	if ord.Code != "" || ord.Period != nil || ord.Despatch != nil || ord.Receiving != nil || ord.Tender != nil || ord.Identities != nil {
		c.inv.Ordering = ord
	}
	return nil
}
