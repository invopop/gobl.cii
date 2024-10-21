package cii

import (
	"github.com/invopop/gobl.cii/structs"
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
	KeyAdditionalDocumentTypeInvoiceDataSheet = "invoice-data-sheet"
	KeyAdditionalDocumentTypeRefPaper         = "ref-paper"
)

func ParseCtoGOrdering(inv *bill.Invoice, doc *structs.XMLDoc) *bill.Ordering {
	ordering := &bill.Ordering{}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerReference != nil {
		ordering.Code = cbc.Code(*doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerReference)
	}

	// Ordering period parsing
	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod != nil {
		period := &cal.Period{}

		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.StartDateTime != nil {
			period.Start = ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.StartDateTime.DateTimeString)
		}

		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.EndDateTime != nil {
			period.End = ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.EndDateTime.DateTimeString)
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.Description != nil {
			period.Label = *doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.Description
		}
		ordering.Period = period
	}

	// Despatch, Receiving and Tender parsing
	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument != nil {
		ordering.Despatch = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.IssuerAssignedID),
			},
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.FormattedIssueDateTime != nil {
			refDate := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.FormattedIssueDateTime.DateTimeString)
			ordering.Despatch[0].IssueDate = &refDate
		}
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument != nil {
		ordering.Receiving = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.IssuerAssignedID),
			},
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.FormattedIssueDateTime != nil {
			refDate := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.FormattedIssueDateTime.DateTimeString)
			ordering.Receiving[0].IssueDate = &refDate
		}
	}

	if len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.AdditionalReferencedDocument) > 0 {
		for _, ref := range doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.AdditionalReferencedDocument {
			switch ref.TypeCode {
			case AdditionalDocumentTypeTender:
				if ordering.Tender == nil {
					ordering.Tender = make([]*org.DocumentRef, 0)
				}
				docRef := &org.DocumentRef{
					Code: cbc.Code(ref.IssuerAssignedID),
				}
				if ref.FormattedIssueDateTime != nil {
					refDate := ParseDate(ref.FormattedIssueDateTime.DateTimeString)
					docRef.IssueDate = &refDate
				}
				ordering.Tender = append(ordering.Tender, docRef)
			case AdditionalDocumentTypeProductInvoice:
				if ordering.Identities == nil {
					ordering.Identities = make([]*org.Identity, 0)
				}
				ordering.Identities = append(ordering.Identities, &org.Identity{
					Key:  KeyAdditionalDocumentTypeInvoiceDataSheet,
					Code: cbc.Code(ref.IssuerAssignedID),
				})
			default:
				ordering.Identities = append(ordering.Identities, &org.Identity{
					Key:  KeyAdditionalDocumentTypeRefPaper,
					Code: cbc.Code(ref.IssuerAssignedID),
				})
			}
		}
	}

	if ordering.Code != "" || ordering.Period != nil || ordering.Despatch != nil || ordering.Receiving != nil || ordering.Tender != nil || ordering.Identities != nil {
		return ordering
	}
	return nil
}
