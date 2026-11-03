package cii

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/xmlctx"
)

// Invoice is a pseudo-model for containing the XML document being created
type Invoice struct {
	XMLName           xml.Name          `xml:"rsm:CrossIndustryInvoice"`
	RSMNamespace      string            `xml:"xmlns:rsm,attr"`
	RAMNamespace      string            `xml:"xmlns:ram,attr"`
	QDTNamespace      string            `xml:"xmlns:qdt,attr"`
	UDTNamespace      string            `xml:"xmlns:udt,attr"`
	ExchangedContext  *ExchangedContext `xml:"rsm:ExchangedDocumentContext"`
	ExchangedDocument *Header           `xml:"rsm:ExchangedDocument"`
	Transaction       *Transaction      `xml:"rsm:SupplyChainTradeTransaction"`
}

// Transaction defines the structure of the transaction in the CII standard
type Transaction struct {
	Lines      []*Line     `xml:"ram:IncludedSupplyChainTradeLineItem"`
	Agreement  *Agreement  `xml:"ram:ApplicableHeaderTradeAgreement"`
	Delivery   *Delivery   `xml:"ram:ApplicableHeaderTradeDelivery"`
	Settlement *Settlement `xml:"ram:ApplicableHeaderTradeSettlement"`
}

// Tax defines the structure of ApplicableTradeTax of the CII standard
type Tax struct {
	CalculatedAmount      string     `xml:"ram:CalculatedAmount,omitempty"`
	TypeCode              string     `xml:"ram:TypeCode,omitempty"`
	ExemptionReason       string     `xml:"ram:ExemptionReason,omitempty"`
	BasisAmount           string     `xml:"ram:BasisAmount,omitempty"`
	CategoryCode          string     `xml:"ram:CategoryCode,omitempty"`
	ExemptionReasonCode   string     `xml:"ram:ExemptionReasonCode,omitempty"`
	DueDateTypeCode       string     `xml:"ram:DueDateTypeCode,omitempty"`
	TaxPointDate          *IssueDate `xml:"ram:TaxPointDate,omitempty"`
	RateApplicablePercent string     `xml:"ram:RateApplicablePercent,omitempty"`
}

// Date defines date in the UDT structure
type Date struct {
	Value  string `xml:",chardata"`
	Format string `xml:"format,attr,omitempty"`
}

// Note defines note in the RAM structure
type Note struct {
	Content     string `xml:"ram:Content,omitempty"`
	SubjectCode string `xml:"ram:SubjectCode,omitempty"`
}

// decodeInvoice unmarshals CII invoice XML into an Invoice struct.
func decodeInvoice(data []byte) (*Invoice, error) {
	inv := new(Invoice)
	if err := xmlctx.Unmarshal(data, inv, xmlctx.WithNamespaces(
		map[string]string{
			nsPrefixRSM: NamespaceRSM,
			nsPrefixRAM: NamespaceRAM,
			nsPrefixQDT: NamespaceQDT,
			nsPrefixUDT: NamespaceUDT,
		},
	)); err != nil {
		return nil, fmt.Errorf("error unmarshaling CII invoice: %w", err)
	}
	return inv, nil
}

// contextIDs provides the guideline (BT-24) and business process (BT-23) IDs
// the invoice declares.
func (out *Invoice) contextIDs() (string, string) {
	var guidelineID, businessID string
	if ec := out.ExchangedContext; ec != nil {
		if ec.GuidelineContext != nil {
			guidelineID = ec.GuidelineContext.ID
		}
		if ec.BusinessContext != nil {
			businessID = ec.BusinessContext.ID
		}
	}
	return guidelineID, businessID
}

func newInvoice(inv *bill.Invoice, f Format) (*Invoice, error) {
	// Determine GuidelineID to use in output
	guidelineID := f.GuidelineID
	if f.OutputGuidelineID != "" {
		guidelineID = f.OutputGuidelineID
	}
	businessID := f.BusinessID

	out := &Invoice{
		RSMNamespace: NamespaceRSM,
		RAMNamespace: NamespaceRAM,
		QDTNamespace: NamespaceQDT,
		UDTNamespace: NamespaceUDT,
		ExchangedContext: &ExchangedContext{
			GuidelineContext: &ExchangedContextParameter{ID: guidelineID},
		},
	}
	if businessID != "" {
		out.ExchangedContext.BusinessContext = &ExchangedContextParameter{ID: businessID}
	}

	if err := out.addHeader(inv); err != nil {
		return nil, err
	}

	if err := out.addTransaction(inv); err != nil {
		return nil, err
	}

	return out, nil
}

// addTransaction adds the transaction part of a EN 16931 compliant invoice
func (out *Invoice) addTransaction(inv *bill.Invoice) error {
	out.Transaction = new(Transaction)

	if err := out.addLines(inv); err != nil {
		return err
	}
	if err := out.addAgreement(inv); err != nil {
		return err
	}
	if len(inv.Attachments) > 0 {
		out.addAttachments(inv)
	}
	var err error
	if out.Transaction.Settlement, err = newSettlement(inv); err != nil {
		return err
	}
	out.Transaction.Delivery = newDelivery(inv)
	return nil
}
