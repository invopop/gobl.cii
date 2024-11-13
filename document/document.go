// Package document contains the CII document model to be used in conversion
package document

import (
	"github.com/nbio/xml"
)

// Document is a pseudo-model for containing the XML document being created
type Document struct {
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
	CalculatedAmount      string `xml:"ram:CalculatedAmount,omitempty"`
	TypeCode              string `xml:"ram:TypeCode,omitempty"`
	BasisAmount           string `xml:"ram:BasisAmount,omitempty"`
	CategoryCode          string `xml:"ram:CategoryCode,omitempty"`
	RateApplicablePercent string `xml:"ram:RateApplicablePercent,omitempty"`
}

// Summary defines the structure of SpecifiedTradeSettlementHeaderMonetarySummation of the CII standard
type Summary struct {
	TotalAmount         string          `xml:"ram:LineTotalAmount"`
	Charges             string          `xml:"ram:ChargeTotalAmount,omitempty"`
	Discounts           string          `xml:"ram:AllowanceTotalAmount,omitempty"`
	TaxBasisTotalAmount string          `xml:"ram:TaxBasisTotalAmount"`
	TaxTotalAmount      *TaxTotalAmount `xml:"ram:TaxTotalAmount"`
	GrandTotalAmount    string          `xml:"ram:GrandTotalAmount"`
	DuePayableAmount    string          `xml:"ram:DuePayableAmount"`
}

// TaxTotalAmount defines the structure of the TaxTotalAmount of the CII standard
type TaxTotalAmount struct {
	Amount   string `xml:",chardata"`
	Currency string `xml:"currencyID,attr"`
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
