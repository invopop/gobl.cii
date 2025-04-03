package cii

import (
	"github.com/nbio/xml"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func parseInvoice(data []byte) (*bill.Invoice, error) {
	in := new(Invoice)
	if err := xml.Unmarshal(data, in); err != nil {
		return nil, err
	}
	return goblInvoice(in)
}

func goblInvoice(in *Invoice) (*bill.Invoice, error) {
	out := &bill.Invoice{
		Code:     cbc.Code(in.ExchangedDocument.ID),
		Type:     typeCodeParse(in.ExchangedDocument.TypeCode),
		Currency: currency.Code(in.Transaction.Settlement.Currency),
		Supplier: goblNewParty(in.Transaction.Agreement.Seller),
		Customer: goblNewParty(in.Transaction.Agreement.Buyer),
		Tax: &bill.Tax{
			Ext: tax.Extensions{
				untdid.ExtKeyDocumentType: cbc.Code(in.ExchangedDocument.TypeCode),
			},
		},
	}

	issueDate, err := parseDate(in.ExchangedDocument.IssueDate.DateFormat.Value)
	if err != nil {
		return nil, err
	}
	out.IssueDate = issueDate

	if err = goblAddLines(in.Transaction, out); err != nil {
		return nil, err
	}

	// Payment comprised of terms, means and payee. Check tehre is relevant info in at least one of them to create a payment
	ahts := in.Transaction.Settlement
	if ahts.HasPayment() {
		if out.Payment, err = goblNewPaymentDetails(ahts); err != nil {
			return nil, err
		}
	}

	if len(in.ExchangedDocument.IncludedNote) > 0 {
		out.Notes = make([]*org.Note, 0, len(in.ExchangedDocument.IncludedNote))
		for _, note := range in.ExchangedDocument.IncludedNote {
			n := &org.Note{
				Text: note.Content,
			}
			if note.SubjectCode != "" {
				n.Code = cbc.Code(note.SubjectCode)
			}
			out.Notes = append(out.Notes, n)
		}
	}

	if out.Ordering, err = goblNewOrdering(in); err != nil {
		return nil, err
	}
	if out.Delivery, err = goblNewDeliveryDetails(in.Transaction.Delivery); err != nil {
		return nil, err
	}

	if len(ahts.ReferencedDocument) > 0 {
		out.Preceding = make([]*org.DocumentRef, 0, len(ahts.ReferencedDocument))
		for _, ref := range ahts.ReferencedDocument {
			docRef := &org.DocumentRef{
				Code: cbc.Code(ref.IssuerAssignedID),
			}
			if ref.IssueDate != nil && ref.IssueDate.DateFormat != nil {
				refDate, err := parseDate(ref.IssueDate.DateFormat.Value)
				if err != nil {
					return nil, err
				}
				docRef.IssueDate = &refDate
			}
			out.Preceding = append(out.Preceding, docRef)
		}
	}

	if in.Transaction.Agreement.TaxRepresentative != nil {
		// Move the original seller to the ordering.seller party
		if out.Ordering == nil {
			out.Ordering = new(bill.Ordering)
		}
		out.Ordering.Seller = out.Supplier

		// Overwrite the seller field with the tax representative
		out.Supplier = goblNewParty(in.Transaction.Agreement.TaxRepresentative)
	}

	if len(ahts.AllowanceCharges) > 0 {
		if err = goblAddChargesAndDiscounts(ahts, out); err != nil {
			return nil, err
		}
	}

	return out, nil
}
