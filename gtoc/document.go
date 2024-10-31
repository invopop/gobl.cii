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
	ID           string  `xml:"ram:ID"`
	TypeCode     string  `xml:"ram:TypeCode"`
	IssueDate    *Date   `xml:"ram:IssueDateTime>udt:DateTimeString"`
	IncludedNote []*Note `xml:"ram:IncludedNote,omitempty"`
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
	BuyerReference    string   `xml:"ram:BuyerReference,omitempty"`
	Seller            *Party   `xml:"ram:SellerTradeParty,omitempty"`
	Buyer             *Party   `xml:"ram:BuyerTradeParty,omitempty"`
	TaxRepresentative *Party   `xml:"ram:SellerTaxRepresentativeTradeParty,omitempty"`
	Sales             *string  `xml:"ram:SellerOrderReferencedDocument>ram:IssuerAssignedID,omitempty"`
	Purchase          *string  `xml:"ram:BuyerOrderReferencedDocument>ram:IssuerAssignedID,omitempty"`
	Contract          *string  `xml:"ram:ContractReferencedDocument>ram:IssuerAssignedID,omitempty"`
	Project           *Project `xml:"ram:SpecifiedProcurringProject,omitempty"`
}

// Project defines common architecture of document reference fields in the CII standard
type Project struct {
	ID   string `xml:"ram:ID,omitempty"`
	Name string `xml:"ram:Name,omitempty"`
}

// Delivery defines the structure of ApplicableHeaderTradeDelivery of the CII standard
type Delivery struct {
	Receiver  *Party  `xml:"ram:ShipToTradeParty,omitempty"`
	Event     *Date   `xml:"ram:ActualDeliverySupplyChainEvent>ram:OccurrenceDateTime>udt:DateTimeString,omitempty"`
	Despatch  *string `xml:"ram:DespatchAdviceReferencedDocument>ram:IssuerAssignedID,omitempty"`
	Receiving *string `xml:"ram:ReceivingAdviceReferencedDocument>ram:IssuerAssignedID,omitempty"`
}

// Settlement defines the structure of ApplicableHeaderTradeSettlement of the CII standard
type Settlement struct {
	CreditorRefID      string              `xml:"ram:CreditorReferenceID,omitempty"`
	Currency           string              `xml:"ram:InvoiceCurrencyCode"`
	Payee              *Party              `xml:"ram:PayeeTradeParty,omitempty"`
	PaymentMeans       *PaymentMeans       `xml:"ram:SpecifiedTradeSettlementPaymentMeans"`
	Tax                []*Tax              `xml:"ram:ApplicableTradeTax"`
	Period             *Period             `xml:"ram:BillingSpecifiedPeriod,omitempty"`
	AllowanceCharges   []*AllowanceCharge  `xml:"ram:SpecifiedTradeAllowanceCharge,omitempty"`
	PaymentTerms       *Terms              `xml:"ram:SpecifiedTradePaymentTerms,omitempty"`
	Summary            *Summary            `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
	ReferencedDocument *ReferencedDocument `xml:"ram:InvoiceReferencedDocument,omitempty"`
}

// Party defines the structure of the TradePartyType of the CII standard
type Party struct {
	ID                        string                     `xml:"ram:ID,omitempty"`
	Name                      string                     `xml:"ram:Name,omitempty"`
	Description               string                     `xml:"ram:Description,omitempty"`
	LegalOrganization         *LegalOrganization         `xml:"ram:SpecifiedLegalOrganization,omitempty"`
	Contact                   *Contact                   `xml:"ram:DefinedTradeContact,omitempty"`
	PostalTradeAddress        *PostalTradeAddress        `xml:"ram:PostalTradeAddress,omitempty"`
	URIUniversalCommunication *URIUniversalCommunication `xml:"ram:URIUniversalCommunication>ram:URIID,omitempty"`
	SpecifiedTaxRegistration  *SpecifiedTaxRegistration  `xml:"ram:SpecifiedTaxRegistration>ram:ID,omitempty"`
}

// Note defines note in the RAM structure
type Note struct {
	Content     string `xml:"ram:Content,omitempty"`
	SubjectCode string `xml:"ram:SubjectCode,omitempty"`
}

// PostalTradeAddress defines the structure of the PostalTradeAddress of the CII standard
type PostalTradeAddress struct {
	Postcode  string `xml:"ram:PostcodeCode"`
	LineOne   string `xml:"ram:LineOne"`
	LineTwo   string `xml:"ram:LineTwo,omitempty"`
	City      string `xml:"ram:CityName"`
	CountryID string `xml:"ram:CountryID"`
	Region    string `xml:"ram:CountrySubDivisionName,omitempty"`
}

// URIUniversalCommunication defines the structure of URIUniversalCommunication of the CII standard
type URIUniversalCommunication struct {
	URIID    string `xml:",chardata"`
	SchemeID string `xml:"schemeID,attr"`
}

// Line defines the structure of the IncludedSupplyChainTradeLineItem in the CII standard
type Line struct {
	ID              string           `xml:"ram:AssociatedDocumentLineDocument>ram:LineID"`
	Note            []Note           `xml:"ram:AssociatedDocumentLineDocument>ram:IncludedNote,omitempty"`
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
	ApplicableTradeTax []*Tax             `xml:"ram:ApplicableTradeTax"`
	AllowanceCharge    []*AllowanceCharge `xml:"ram:SpecifiedTradeAllowanceCharge,omitempty"`
	Sum                string             `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation>ram:LineTotalAmount"`
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
	PersonName string       `xml:"ram:PersonName,omitempty"`
	Phone      *PhoneNumber `xml:"ram:TelephoneUniversalCommunication,omitempty"`
	Email      *Email       `xml:"ram:EmailURIUniversalCommunication,omitempty"`
}

// PhoneNumber defines the structure of the TelephoneUniversalCommunication of the CII standard
type PhoneNumber struct {
	CompleteNumber string `xml:"ram:CompleteNumber,omitempty"`
}

// Email defines the structure of the EmailURIUniversalCommunication of the CII standard
type Email struct {
	URIID string `xml:"ram:URIID,omitempty"`
}

// Terms defines the structure of SpecifiedTradePaymentTerms of the CII standard
type Terms struct {
	Description string `xml:"ram:Description,omitempty"`
	Mandate     string `xml:"ram:DirectDebitMandateID,omitempty"`
}

// PaymentMeans defines the structure of SpecifiedTradeSettlementPaymentMeans of the CII standard
type PaymentMeans struct {
	TypeCode    string    `xml:"ram:TypeCode"`
	Information string    `xml:"ram:Information,omitempty"`
	Card        *Card     `xml:"ram:ApplicableTradeSettlementFinancialCard,omitempty"`
	Debtor      string    `xml:"ram:PayerPartyDebtorFinancialAccount>ram:IBANID,omitempty"`
	Creditor    *Creditor `xml:"ram:PayeePartyCreditorFinancialAccount,omitempty"`
	BICID       string    `xml:"ram:PayeeSpecifiedCreditorFinancialInstitution>ram:BICID,omitempty"`
}

// Creditor defines the structure of PayeePartyCreditorFinancialAccount of the CII standard
type Creditor struct {
	IBAN   string `xml:"ram:IBANID,omitempty"`
	Name   string `xml:"ram:AccountName,omitempty"`
	Number string `xml:"ram:ProprietaryID,omitempty"`
}

// Card defines the structure of ApplicableTradeSettlementFinancialCard of the CII standard
type Card struct {
	ID   string `xml:"ram:ID,omitempty"`
	Name string `xml:"ram:CardholderName,omitempty"`
}

// Tax defines the structure of ApplicableTradeTax of the CII standard
type Tax struct {
	CalculatedAmount      string `xml:"ram:CalculatedAmount,omitempty"`
	TypeCode              string `xml:"ram:TypeCode,omitempty"`
	BasisAmount           string `xml:"ram:BasisAmount,omitempty"`
	CategoryCode          string `xml:"ram:CategoryCode,omitempty"`
	RateApplicablePercent string `xml:"ram:RateApplicablePercent,omitempty"`
}

// AllowanceCharge defines the structure of SpecifiedTradeAllowanceCharge of the CII standard, also used for line items
type AllowanceCharge struct {
	ChargeIndicator Indicator `xml:"ram:ChargeIndicator"`
	Percent         string    `xml:"ram:CalculationPercent,omitempty"`
	Base            string    `xml:"ram:BasisAmount,omitempty"`
	Amount          string    `xml:"ram:ActualAmount,omitempty"`
	ReasonCode      string    `xml:"ram:ReasonCode,omitempty"`
	Reason          string    `xml:"ram:Reason,omitempty"`
	Tax             *Tax      `xml:"ram:CategoryTradeTax,omitempty"`
}

// Indicator defines the structure of Indicator of the CII standard
type Indicator struct {
	Value bool `xml:"udt:Indicator"`
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

// Period defines the structure of the ExpectedDeliveryPeriod of the CII standard
type Period struct {
	Start *Date `xml:"ram:StartDateTime>udt:DateTimeString"`
	End   *Date `xml:"ram:EndDateTime>udt:DateTimeString"`
}

// Date defines date in the UDT structure
type Date struct {
	Date   string `xml:",chardata"`
	Format string `xml:"format,attr,omitempty"`
}
