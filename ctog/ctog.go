package ctog

import (
	"encoding/xml"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
)

// Conversor is a struct that contains the necessary elements to convert between GOBL and CII
type Conversor struct {
	// CtoG Output
	inv *bill.Invoice
	// CtoG Input
	doc *Document
}

// Initialize a new Conversor
func NewConversor() *Conversor {
	c := new(Conversor)
	c.inv = new(bill.Invoice)
	c.doc = new(Document)
	return c
}

// GetInvoice returns the invoice from the conversor
func (c *Conversor) GetInvoice() *bill.Invoice {
	return c.inv
}

func (c *Conversor) ConvertToGOBL(xmlData []byte) (*gobl.Envelope, error) {
	if err := xml.Unmarshal(xmlData, &c.doc); err != nil {
		return nil, err
	}

	inv, err := c.NewInvoice(c.doc)

	if err != nil {
		return nil, err
	}
	env, err := gobl.Envelop(inv)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (c *Conversor) NewInvoice(doc *Document) (*bill.Invoice, error) {

	c.inv = &bill.Invoice{
		Code:      cbc.Code(doc.ExchangedDocument.ID),
		Type:      TypeCodeParse(doc.ExchangedDocument.TypeCode),
		IssueDate: ParseDate(doc.ExchangedDocument.IssueDateTime.DateTimeString.Value),
		Currency:  currency.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.InvoiceCurrencyCode),
		Supplier:  c.getParty(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTradeParty),
		Customer:  c.getParty(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerTradeParty),
	}

	c.getLines(&doc.SupplyChainTradeTransaction)

	// Payment comprised of terms, means and payee. Check tehre is relevant info in at least one of them to create a payment
	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.PayeeTradeParty != nil ||
		(len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradePaymentTerms) > 0 &&
			doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradePaymentTerms[0].DueDateDateTime != nil) ||
		(len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeSettlementPaymentMeans) > 0 &&
			doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeSettlementPaymentMeans[0].TypeCode != "1") {
		c.getPayment(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement)
	}

	if len(doc.ExchangedDocument.IncludedNote) > 0 {
		c.inv.Notes = make([]*cbc.Note, 0, len(doc.ExchangedDocument.IncludedNote))
		for _, note := range doc.ExchangedDocument.IncludedNote {
			n := &cbc.Note{
				Text: note.Content,
			}
			if note.SubjectCode != "" {
				n.Code = note.SubjectCode
			}
			c.inv.Notes = append(c.inv.Notes, n)
		}
	}

	err := c.getOrdering(doc)
	if err != nil {
		return nil, err
	}

	err = c.getDelivery(doc)
	if err != nil {
		return nil, err
	}

	if len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.InvoiceReferencedDcument) > 0 {
		c.inv.Preceding = make([]*org.DocumentRef, 0, len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.InvoiceReferencedDcument))
		for _, ref := range doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.InvoiceReferencedDcument {
			docRef := &org.DocumentRef{
				Code: cbc.Code(ref.IssuerAssignedID),
			}
			if ref.FormattedIssueDateTime != nil {
				refDate := ParseDate(ref.FormattedIssueDateTime.DateTimeString.Value)
				docRef.IssueDate = &refDate
			}
			c.inv.Preceding = append(c.inv.Preceding, docRef)
		}
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTaxRepresentativeTradeParty != nil {
		// Move the original seller to the ordering.seller party
		if c.inv.Ordering == nil {
			c.inv.Ordering = &bill.Ordering{}
		}
		c.inv.Ordering.Seller = c.inv.Supplier

		// Overwrite the seller field with the tax representative
		c.inv.Supplier = c.getParty(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTaxRepresentativeTradeParty)
	}

	if len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeAllowanceCharge) > 0 {
		err = c.getCharges(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement)
		if err != nil {
			return nil, err
		}
	}

	return c.inv, nil
}
