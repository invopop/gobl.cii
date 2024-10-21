// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package cii

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl"
	ctog "github.com/invopop/gobl.cii/internal/ctog"
	gtoc "github.com/invopop/gobl.cii/internal/gtoc"
	"github.com/invopop/gobl.cii/structs"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
)

// CFDI schema constants
const (
	RSM              = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	RAM              = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	QDT              = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
	UDT              = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
	BusinessProcess  = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
	GuidelineContext = "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0"
)

// Document is a pseudo-model for containing the XML document being created
type Document struct {
	XMLName                xml.Name          `xml:"rsm:CrossIndustryInvoice"`
	RSMNamespace           string            `xml:"xmlns:rsm,attr"`
	RAMNamespace           string            `xml:"xmlns:ram,attr"`
	QDTNamespace           string            `xml:"xmlns:qdt,attr"`
	UDTNamespace           string            `xml:"xmlns:udt,attr"`
	BusinessProcessContext string            `xml:"rsm:ExchangedDocumentContext>ram:BusinessProcessSpecifiedDocumentContextParameter>ram:ID"`
	GuidelineContext       string            `xml:"rsm:ExchangedDocumentContext>ram:GuidelineSpecifiedDocumentContextParameter>ram:ID"`
	ExchangedDocument      *gtoc.Header      `xml:"rsm:ExchangedDocument"`
	Transaction            *gtoc.Transaction `xml:"rsm:SupplyChainTradeTransaction"`
}

// NewDocument converts a GOBL envelope into a XRechnung and Factur-X document
func NewDocument(env *gobl.Envelope) (*Document, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}

	transaction, err := gtoc.NewTransaction(inv)
	if err != nil {
		return nil, err
	}

	doc := Document{
		RSMNamespace:           RSM,
		RAMNamespace:           RAM,
		QDTNamespace:           QDT,
		UDTNamespace:           UDT,
		BusinessProcessContext: BusinessProcess,
		GuidelineContext:       GuidelineContext,
		ExchangedDocument:      gtoc.NewHeader(inv),
		Transaction:            transaction,
	}
	return &doc, nil
}

// Bytes returns the XML representation of the document in bytes
func (d *Document) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}

// NewDocument converts a XRechnung document into a GOBL envelope
func NewGOBLFromCII(doc *structs.XMLDoc) (*gobl.Envelope, error) {

	inv := mapCIIToInvoice(doc)
	env, err := gobl.Envelop(inv)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func mapCIIToInvoice(doc *structs.XMLDoc) *bill.Invoice {

	inv := &bill.Invoice{
		Code:      cbc.Code(doc.ExchangedDocument.ID),
		Type:      ctog.TypeCodeParse(doc.ExchangedDocument.TypeCode),
		IssueDate: ctog.ParseDate(doc.ExchangedDocument.IssueDateTime.DateTimeString.Value),
		Currency:  currency.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.InvoiceCurrencyCode),
		Supplier:  ctog.ParseCtoGParty(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTradeParty),
		Customer:  ctog.ParseCtoGParty(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerTradeParty),
		Lines:     ctog.ParseCtoGLines(&doc.SupplyChainTradeTransaction),
	}

	// Payment comprised of terms, means and payee. Check tehre is relevant info in at least one of them to create a payment
	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.PayeeTradeParty != nil ||
		(len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradePaymentTerms) > 0 &&
			doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradePaymentTerms[0].DueDateDateTime != nil) ||
		(len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeSettlementPaymentMeans) > 0 &&
			doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeSettlementPaymentMeans[0].TypeCode != "1") {
		inv.Payment = ctog.ParseCtoGPayment(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement)
	}

	if len(doc.ExchangedDocument.IncludedNote) > 0 {
		inv.Notes = make([]*cbc.Note, 0, len(doc.ExchangedDocument.IncludedNote))
		for _, note := range doc.ExchangedDocument.IncludedNote {
			n := &cbc.Note{
				Text: note.Content,
			}
			if note.SubjectCode != "" {
				n.Code = note.SubjectCode
			}
			inv.Notes = append(inv.Notes, n)
		}
	}

	ordering := ctog.ParseCtoGOrdering(inv, doc)
	if ordering != nil {
		inv.Ordering = ordering
	}

	delivery := ctog.ParseCtoGDelivery(inv, doc)
	if delivery != nil {
		inv.Delivery = delivery
	}

	if len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.InvoiceReferencedDcument) > 0 {
		inv.Preceding = make([]*org.DocumentRef, 0, len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.InvoiceReferencedDcument))
		for _, ref := range doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.InvoiceReferencedDcument {
			docRef := &org.DocumentRef{
				Code: cbc.Code(ref.IssuerAssignedID),
			}
			if ref.FormattedIssueDateTime != nil {
				refDate := ctog.ParseDate(ref.FormattedIssueDateTime.DateTimeString.Value)
				docRef.IssueDate = &refDate
			}
			inv.Preceding = append(inv.Preceding, docRef)
		}
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTaxRepresentativeTradeParty != nil {
		// Move the original seller to the ordering.seller party
		if inv.Ordering == nil {
			inv.Ordering = &bill.Ordering{}
		}
		inv.Ordering.Seller = inv.Supplier

		// Overwrite the seller field with the tax representative
		inv.Supplier = ctog.ParseCtoGParty(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTaxRepresentativeTradeParty)
	}

	if len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeAllowanceCharge) > 0 {
		charges, discounts := ctog.ParseCtoGCharges(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement)
		if len(charges) > 0 {
			inv.Charges = charges
		}
		if len(discounts) > 0 {
			inv.Discounts = discounts
		}
	}

	return inv
}
