package document

// Settlement defines the structure of ApplicableHeaderTradeSettlement of the CII standard
type Settlement struct {
	CreditorRefID      string                `xml:"ram:CreditorReferenceID,omitempty"`
	Currency           string                `xml:"ram:InvoiceCurrencyCode"`
	Payee              *Party                `xml:"ram:PayeeTradeParty,omitempty"`
	PaymentMeans       []*PaymentMeans       `xml:"ram:SpecifiedTradeSettlementPaymentMeans"`
	Tax                []*Tax                `xml:"ram:ApplicableTradeTax"`
	Period             *Period               `xml:"ram:BillingSpecifiedPeriod,omitempty"`
	AllowanceCharges   []*AllowanceCharge    `xml:"ram:SpecifiedTradeAllowanceCharge,omitempty"`
	PaymentTerms       []*Terms              `xml:"ram:SpecifiedTradePaymentTerms,omitempty"`
	Summary            *Summary              `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
	ReferencedDocument []*ReferencedDocument `xml:"ram:InvoiceReferencedDocument,omitempty"`
	Advance            []*Advance            `xml:"ram:SpecifiedAdvancePayment,omitempty"`
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

// Terms defines the structure of SpecifiedTradePaymentTerms of the CII standard
type Terms struct {
	Description    string     `xml:"ram:Description,omitempty"`
	DueDate        *IssueDate `xml:"ram:DueDateDateTime,omitempty"`
	Mandate        string     `xml:"ram:DirectDebitMandateID,omitempty"`
	PartialPayment string     `xml:"ram:PartialPaymentAmount,omitempty"`
}

// PaymentMeans defines the structure of SpecifiedTradeSettlementPaymentMeans of the CII standard
type PaymentMeans struct {
	TypeCode            string               `xml:"ram:TypeCode"`
	Information         string               `xml:"ram:Information,omitempty"`
	Card                *Card                `xml:"ram:ApplicableTradeSettlementFinancialCard,omitempty"`
	Debtor              *DebtorAccount       `xml:"ram:PayerPartyDebtorFinancialAccount,omitempty"`
	Creditor            *Creditor            `xml:"ram:PayeePartyCreditorFinancialAccount,omitempty"`
	CreditorInstitution *CreditorInstitution `xml:"ram:PayeeSpecifiedCreditorFinancialInstitution,omitempty"`
}

// DebtorAccount defines the structure of PayerPartyDebtorFinancialAccount of the CII standard
type DebtorAccount struct {
	IBAN string `xml:"ram:IBANID,omitempty"`
}

// Creditor defines the structure of PayeePartyCreditorFinancialAccount of the CII standard
type Creditor struct {
	IBAN   string `xml:"ram:IBANID,omitempty"`
	Name   string `xml:"ram:AccountName,omitempty"`
	Number string `xml:"ram:ProprietaryID,omitempty"`
}

// CreditorInstitution defines the structure of PayeeSpecifiedCreditorFinancialInstitution of the CII standard
type CreditorInstitution struct {
	BIC string `xml:"ram:BICID,omitempty"`
}

// Card defines the structure of ApplicableTradeSettlementFinancialCard of the CII standard
type Card struct {
	ID   string `xml:"ram:ID,omitempty"`
	Name string `xml:"ram:CardholderName,omitempty"`
}

// Advance defines the structure of SpecifiedAdvancePayment of the CII standard
type Advance struct {
	Amount string              `xml:"ram:PaidAmount"`
	Date   *FormattedIssueDate `xml:"ram:FormattedReceivedDateTime,omitempty"`
}

// ReferencedDocument defines the structure of InvoiceReferencedDocument of the CII standard
type ReferencedDocument struct {
	IssuerAssignedID string              `xml:"ram:IssuerAssignedID,omitempty"`
	IssueDate        *FormattedIssueDate `xml:"ram:FormattedIssueDateTime,omitempty"`
}

// Period defines the structure of the ExpectedDeliveryPeriod of the CII standard
type Period struct {
	Description *string    `xml:"ram:Description,omitempty"`
	Start       *IssueDate `xml:"ram:StartDateTime"`
	End         *IssueDate `xml:"ram:EndDateTime"`
}
