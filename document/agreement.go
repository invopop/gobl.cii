package document

// Agreement defines the structure of the ApplicableHeaderTradeAgreement of the CII standard
type Agreement struct {
	BuyerReference     string                `xml:"ram:BuyerReference,omitempty"`
	Seller             *Party                `xml:"ram:SellerTradeParty,omitempty"`
	Buyer              *Party                `xml:"ram:BuyerTradeParty,omitempty"`
	TaxRepresentative  *Party                `xml:"ram:SellerTaxRepresentativeTradeParty,omitempty"`
	Sales              *IssuerID             `xml:"ram:SellerOrderReferencedDocument,omitempty"`
	Purchase           *IssuerID             `xml:"ram:BuyerOrderReferencedDocument,omitempty"`
	Contract           *IssuerID             `xml:"ram:ContractReferencedDocument,omitempty"`
	AdditionalDocument []*AdditionalDocument `xml:"ram:AdditionalReferencedDocument,omitempty"`
	Project            *Project              `xml:"ram:SpecifiedProcurringProject,omitempty"`
}

// Project defines common architecture of document reference fields in the CII standard
type Project struct {
	ID   string `xml:"ram:ID,omitempty"`
	Name string `xml:"ram:Name,omitempty"`
}

// AdditionalDocument defines the structure of AdditionalReferencedDocument of the CII standard
type AdditionalDocument struct {
	ID        string              `xml:"ram:IssuerAssignedID,omitempty"`
	TypeCode  string              `xml:"ram:TypeCode,omitempty"`
	Name      string              `xml:"ram:Name,omitempty"`
	IssueDate *FormattedIssueDate `xml:"ram:FormattedIssueDateTime,omitempty"`
}

// IssuerID defines the structure of IssuerAssignedID of the CII standard
type IssuerID struct {
	ID string `xml:"ram:IssuerAssignedID,omitempty"`
}
