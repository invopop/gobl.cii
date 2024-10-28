package gtoc

import (
	"encoding/xml"
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
	XMLName                xml.Name     `xml:"rsm:CrossIndustryInvoice"`
	RSMNamespace           string       `xml:"xmlns:rsm,attr"`
	RAMNamespace           string       `xml:"xmlns:ram,attr"`
	QDTNamespace           string       `xml:"xmlns:qdt,attr"`
	UDTNamespace           string       `xml:"xmlns:udt,attr"`
	BusinessProcessContext string       `xml:"rsm:ExchangedDocumentContext>ram:BusinessProcessSpecifiedDocumentContextParameter>ram:ID"`
	GuidelineContext       string       `xml:"rsm:ExchangedDocumentContext>ram:GuidelineSpecifiedDocumentContextParameter>ram:ID"`
	ExchangedDocument      *Header      `xml:"rsm:ExchangedDocument"`
	Transaction            *Transaction `xml:"rsm:SupplyChainTradeTransaction"`
}

// Header a collection of data for a Cross Industry Invoice Header that is exchanged between two or more parties in written, printed or electronic form.
type Header struct {
	ID           string `xml:"ram:ID"`
	TypeCode     string `xml:"ram:TypeCode"`
	IssueDate    *Date  `xml:"ram:IssueDateTime>udt:DateTimeString"`
	IncludedNote *Note  `xml:"ram:IncludedNote,omitempty"`
}

// Transaction defines the structure of the transaction in the CII standard
type Transaction struct {
	Lines      []*Line     `xml:"ram:IncludedSupplyChainTradeLineItem"`
	Agreement  *Agreement  `xml:"ram:ApplicableHeaderTradeAgreement"`
	Delivery   *Delivery   `xml:"ram:ApplicableHeaderTradeDelivery"`
	Settlement *Settlement `xml:"ram:ApplicableHeaderTradeSettlement"`
}

// Agreement defines the structure of the ApplicableHeaderTradeAgreement of the CII standard
type Agreement struct {
	BuyerReference string  `xml:"ram:BuyerReference,omitempty"`
	Seller         *Seller `xml:"ram:SellerTradeParty,omitempty"`
	Buyer          *Buyer  `xml:"ram:BuyerTradeParty,omitempty"`
	ProjectID      string  `xml:"ram:SpecifiedProcurringProject>ram:ID,omitempty"`
	ProjectName    string  `xml:"ram:SpecifiedProcurringProject>ram:Name,omitempty"`
	Contract       string  `xml:"ram:ContractReferencedDocument>ram:IssuerAssignedID,omitempty"`
	Purchase       string  `xml:"ram:BuyerOrderReferencedDocument>ram:IssuerAssignedID,omitempty"`
	Sales          string  `xml:"ram:SellerOrderReferencedDocument>ram:IssuerAssignedID,omitempty"`
	Receiving      string  `xml:"ram:ReceivingAdviceReferencedDocument>ram:IssuerAssignedID,omitempty"`
	Despatch       string  `xml:"ram:DespatchAdviceReferencedDocument>ram:IssuerAssignedID,omitempty"`
	// Tender         string  `xml:"ram:AdditionalReferencedDocument>ram:IssuerAssignedID,omitempty"` // this can be bt-17, bt-18
}

// Delivery defines the structure of ApplicableHeaderTradeDelivery of the CII standard
type Delivery struct {
	Event *Date `xml:"ram:ActualDeliverySupplyChainEvent>ram:OccurrenceDateTime>udt:DateTimeString,omitempty"`
}

// Settlement defines the structure of ApplicableHeaderTradeSettlement of the CII standard
type Settlement struct {
	Currency           string              `xml:"ram:InvoiceCurrencyCode"`
	TypeCode           string              `xml:"ram:SpecifiedTradeSettlementPaymentMeans>ram:TypeCode"`
	Tax                []*Tax              `xml:"ram:ApplicableTradeTax"`
	PaymentTerms       string              `xml:"ram:SpecifiedTradePaymentTerms>ram:Description,omitempty"`
	Summary            *Summary            `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
	ReferencedDocument *ReferencedDocument `xml:"ram:InvoiceReferencedDocument,omitempty"`
}

// Seller defines the structure of the SellerTradeParty of the CII standard
type Seller struct {
	Name                      string                     `xml:"ram:Name"`
	LegalOrganization         *LegalOrganization         `xml:"ram:SpecifiedLegalOrganization,omitempty"`
	Contact                   *Contact                   `xml:"ram:DefinedTradeContact"`
	PostalTradeAddress        *PostalTradeAddress        `xml:"ram:PostalTradeAddress"`
	URIUniversalCommunication *URIUniversalCommunication `xml:"ram:URIUniversalCommunication>ram:URIID"`
	SpecifiedTaxRegistration  *SpecifiedTaxRegistration  `xml:"ram:SpecifiedTaxRegistration>ram:ID"`
}

// Buyer defines the structure of the BuyerTradeParty of the CII standard
type Buyer struct {
	ID                        string                     `xml:"ram:ID,omitempty"`
	Name                      string                     `xml:"ram:Name"`
	PostalTradeAddress        *PostalTradeAddress        `xml:"ram:PostalTradeAddress"`
	URIUniversalCommunication *URIUniversalCommunication `xml:"ram:URIUniversalCommunication>ram:URIID"`
}

// Note defines note in the RAM structure
type Note struct {
	Content     string `xml:"ram:Content"`
	SubjectCode string `xml:"ram:SubjectCode"`
}

// PostalTradeAddress defines the structure of the PostalTradeAddress of the CII standard
type PostalTradeAddress struct {
	Postcode  string `xml:"ram:PostcodeCode"`
	LineOne   string `xml:"ram:LineOne"`
	LineTwo   string `xml:"ram:LineTwo,omitempty"`
	City      string `xml:"ram:CityName"`
	Region    string `xml:"ram:CountrySubDivisionName,omitempty"`
	CountryID string `xml:"ram:CountryID"`
}

// URIUniversalCommunication defines the structure of URIUniversalCommunication of the CII standard
type URIUniversalCommunication struct {
	URIID    string `xml:",chardata"`
	SchemeID string `xml:"schemeID,attr"`
}

// Line defines the structure of the IncludedSupplyChainTradeLineItem in the CII standard
type Line struct {
	ID              string           `xml:"ram:AssociatedDocumentLineDocument>ram:LineID"`
	Name            string           `xml:"ram:SpecifiedTradeProduct>ram:Name"`
	NetPrice        string           `xml:"ram:SpecifiedLineTradeAgreement>ram:NetPriceProductTradePrice>ram:ChargeAmount"`
	TradeDelivery   *Quantity        `xml:"ram:SpecifiedLineTradeDelivery>ram:BilledQuantity"`
	TradeSettlement *TradeSettlement `xml:"ram:SpecifiedLineTradeSettlement"`
}

// Quantity defines the structure of the quantity with its attributes for the CII standard
type Quantity struct {
	Amount   string `xml:",chardata"`
	UnitCode string `xml:"unitCode,attr"`
}

// TradeSettlement defines the structure of the SpecifiedLineTradeSettlement of the CII standard
type TradeSettlement struct {
	ApplicableTradeTax []*ApplicableTradeTax `xml:"ram:ApplicableTradeTax"`
	Sum                string                `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation>ram:LineTotalAmount"`
}

// ApplicableTradeTax defines the structure of ApplicableTradeTax of the CII standard
type ApplicableTradeTax struct {
	TaxType        string `xml:"ram:TypeCode"`
	TaxCode        string `xml:"ram:CategoryCode"`
	TaxRatePercent string `xml:"ram:RateApplicablePercent"`
}

// SpecifiedTaxRegistration defines the structure of the SpecifiedTaxRegistration of the CII standard
type SpecifiedTaxRegistration struct {
	ID       string `xml:",chardata"`
	SchemeID string `xml:"schemeID,attr"`
}

// LegalOrganization defines the structure of the SpecifiedLegalOrganization of the CII standard
type LegalOrganization struct {
	ID   string `xml:"ram:ID"`
	Name string `xml:"ram:TradingBusinessName"`
}

// Contact defines the structure of the DefinedTradeContact of the CII standard
type Contact struct {
	PersonName string `xml:"ram:PersonName"`
	Phone      string `xml:"ram:TelephoneUniversalCommunication>ram:CompleteNumber"`
	Email      string `xml:"ram:EmailURIUniversalCommunication>ram:URIID"`
}

// Tax defines the structure of ApplicableTradeTax of the CII standard
type Tax struct {
	CalculatedAmount      string `xml:"ram:CalculatedAmount"`
	TypeCode              string `xml:"ram:TypeCode"`
	BasisAmount           string `xml:"ram:BasisAmount"`
	CategoryCode          string `xml:"ram:CategoryCode"`
	RateApplicablePercent string `xml:"ram:RateApplicablePercent"`
}

// Summary defines the structure of SpecifiedTradeSettlementHeaderMonetarySummation of the CII standard
type Summary struct {
	TotalAmount         string          `xml:"ram:LineTotalAmount"`
	TaxBasisTotalAmount string          `xml:"ram:TaxBasisTotalAmount"`
	TaxTotalAmount      *TaxTotalAmount `xml:"ram:TaxTotalAmount"`
	GrandTotalAmount    string          `xml:"ram:GrandTotalAmount"`
	DuePayableAmount    string          `xml:"ram:DuePayableAmount"`
}

// ReferencedDocument defines the structure of InvoiceReferencedDocument of the CII standard
type ReferencedDocument struct {
	IssuerAssignedID string `xml:"ram:IssuerAssignedID,omitempty"`
	IssueDate        *Date  `xml:"ram:FormattedIssueDateTime>qdt:DateTimeString,omitempty"`
}

// TaxTotalAmount defines the structure of the TaxTotalAmount of the CII standard
type TaxTotalAmount struct {
	Amount   string `xml:",chardata"`
	Currency string `xml:"currencyID,attr"`
}
