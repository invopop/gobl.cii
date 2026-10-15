package cii

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

func goblNewOrdering(in *Invoice) (*bill.Ordering, error) {
	ord := new(bill.Ordering)
	tr := in.Transaction

	if tr.Agreement.BuyerReference != "" {
		ord.Code = cbc.Code(tr.Agreement.BuyerReference)
	}

	if tr.Settlement.Invoicer != nil {
		ord.Issuer = goblNewParty(tr.Settlement.Invoicer)
	}

	// BT-19: Buyer accounting reference
	if tr.Settlement.AccountingAccount != nil && tr.Settlement.AccountingAccount.ID != "" {
		ord.Cost = cbc.Code(tr.Settlement.AccountingAccount.ID)
	}

	ord.Sales = goblIssuerRefs(tr.Agreement.Sales)

	ord.Purchases = goblIssuerRefs(tr.Agreement.Purchase)

	if tr.Agreement.Project != nil && tr.Agreement.Project.ID != "" {
		ord.Projects = []*org.DocumentRef{
			{
				Code:        cbc.Code(tr.Agreement.Project.ID),
				Description: cleanString(tr.Agreement.Project.Name),
			},
		}
	}

	ord.Contracts = goblIssuerRefs(tr.Agreement.Contract)

	// Ordering period parsing
	if tr.Settlement.Period != nil {
		per := &cal.Period{}

		if tr.Settlement.Period.Start != nil && tr.Settlement.Period.Start.DateFormat != nil {
			start, err := parseDate(tr.Settlement.Period.Start.DateFormat.Value)
			if err != nil {
				return nil, err
			}
			per.Start = &start
		}

		if tr.Settlement.Period.End != nil && tr.Settlement.Period.End.DateFormat != nil {
			end, err := parseDate(tr.Settlement.Period.End.DateFormat.Value)
			if err != nil {
				return nil, err
			}
			per.End = &end
		}
		if tr.Settlement.Period.Description != nil {
			per.Label = cleanString(*tr.Settlement.Period.Description)
		}
		if per.Start != nil || per.End != nil || per.Label != "" {
			ord.Period = per
		}
	}

	// Despatch, Receiving and Tender parsing
	ord.Despatch = goblIssuerRefs(tr.Delivery.Despatch)

	ord.Receiving = goblIssuerRefs(tr.Delivery.Receiving)

	if len(tr.Agreement.AdditionalDocument) > 0 {
		for _, ref := range tr.Agreement.AdditionalDocument {
			if ref.ID == "" {
				continue
			}
			switch ref.TypeCode {
			case AdditionalDocumentTypeTender:
				if ord.Tender == nil {
					ord.Tender = make([]*org.DocumentRef, 0)
				}
				docRef := &org.DocumentRef{
					Code: cbc.Code(ref.ID),
				}
				if ref.IssueDate != nil && ref.IssueDate.DateFormat != nil {
					refDate, err := parseDate(ref.IssueDate.DateFormat.Value)
					if err != nil {
						return nil, err
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

	if goblOrderingHasData(ord) {
		return ord, nil
	}
	return nil, nil
}

func goblIssuerRefs(ref *IssuerID) []*org.DocumentRef {
	if ref == nil || ref.ID == "" {
		return nil
	}
	return []*org.DocumentRef{{Code: cbc.Code(ref.ID)}}
}

func goblOrderingHasData(ord *bill.Ordering) bool {
	return ord.Code != "" ||
		ord.Period != nil ||
		ord.Despatch != nil ||
		ord.Receiving != nil ||
		ord.Tender != nil ||
		ord.Identities != nil ||
		ord.Sales != nil ||
		ord.Purchases != nil ||
		ord.Projects != nil ||
		ord.Contracts != nil ||
		ord.Issuer != nil ||
		ord.Buyer != nil
}
