package ctog

import (
	"encoding/xml"
)

//This file builds the strcuture of a CII document based on the 16B schema, compliant
//with the EN 16931 standard. Field names are verbatim from the schema and can be found at
//https://unece.org/trade/uncefact/xml-schemas-2018-2012

// Model for XML excluding namespaces
type Document struct {
	XMLName                     xml.Name                    `xml:"CrossIndustryInvoice"`
	BusinessProcessContext      string                      `xml:"ExchangedDocumentContext>BusinessProcessSpecifiedDocumentContextParameter>ID"`
	GuidelineContext            string                      `xml:"ExchangedDocumentContext>GuidelineSpecifiedDocumentContextParameter>ID"`
	ExchangedDocument           ExchangedDocument           `xml:"ExchangedDocument"`
	SupplyChainTradeTransaction SupplyChainTradeTransaction `xml:"SupplyChainTradeTransaction"`
}

type ExchangedDocument struct {
	ID            string `xml:"ID"`
	Name          string `xml:"Name"`
	TypeCode      string `xml:"TypeCode"`
	IssueDateTime struct {
		DateTimeString struct {
			Value  string `xml:",chardata"`
			Format string `xml:"format,attr"`
		} `xml:"DateTimeString"`
	} `xml:"IssueDateTime"`
	IncludedNote []IncludedNote `xml:"IncludedNote"`
}

type SupplyChainTradeTransaction struct {
	IncludedSupplyChainTradeLineItem []IncludedSupplyChainTradeLineItem `xml:"IncludedSupplyChainTradeLineItem"`
	ApplicableHeaderTradeAgreement   ApplicableHeaderTradeAgreement     `xml:"ApplicableHeaderTradeAgreement"`
	ApplicableHeaderTradeDelivery    *ApplicableHeaderTradeDelivery     `xml:"ApplicableHeaderTradeDelivery"`
	ApplicableHeaderTradeSettlement  ApplicableHeaderTradeSettlement    `xml:"ApplicableHeaderTradeSettlement"`
}

type IncludedSupplyChainTradeLineItem struct {
	AssociatedDocumentLineDocument struct {
		LineID       int            `xml:"LineID"`
		IncludedNote []IncludedNote `xml:"IncludedNote"`
	} `xml:"AssociatedDocumentLineDocument"`
	SpecifiedTradeProduct struct {
		Name             string  `xml:"Name"`
		Description      *string `xml:"Description"`
		SellerAssignedID *string `xml:"SellerAssignedID"`
		BuyerAssignedID  *string `xml:"BuyerAssignedID"`
		GlobalID         *struct {
			Value    string `xml:",chardata"`
			SchemeID string `xml:"schemeID,attr"`
		} `xml:"GlobalID"`
		OriginTradeCountry *struct {
			ID string `xml:"ID"`
		} `xml:"OriginTradeCountry"`
		DesignatedProductClassification []*struct {
			ClassName *string `xml:"ClassName"`
			ClassCode struct {
				Value         string `xml:",chardata"`
				ListID        string `xml:"ListID,attr"`
				ListVersionID string `xml:"ListVersionID,attr"`
			} `xml:"ClassCode"`
		} `xml:"DesignatedProductClassification"`
	} `xml:"SpecifiedTradeProduct"`
	SpecifiedLineTradeAgreement struct {
		NetPriceProductTradePrice struct {
			ChargeAmount string `xml:"ChargeAmount"`
		} `xml:"NetPriceProductTradePrice"`
	} `xml:"SpecifiedLineTradeAgreement"`
	SpecifiedLineTradeDelivery *struct {
		BilledQuantity struct {
			Value    string `xml:",chardata"`
			UnitCode string `xml:"unitCode,attr"`
		} `xml:"BilledQuantity"`
	} `xml:"SpecifiedLineTradeDelivery"`
	SpecifiedLineTradeSettlement struct {
		ApplicableTradeTax struct {
			TypeCode              string `xml:"TypeCode"`
			CategoryCode          string `xml:"CategoryCode"`
			RateApplicablePercent string `xml:"RateApplicablePercent"`
		} `xml:"ApplicableTradeTax"`
		SpecifiedTradeAllowanceCharge                 []*SpecifiedTradeAllowanceCharge `xml:"SpecifiedTradeAllowanceCharge"`
		SpecifiedTradeSettlementLineMonetarySummation struct {
			LineTotalAmount float64 `xml:"LineTotalAmount"`
		} `xml:"SpecifiedTradeSettlementLineMonetarySummation"`
		ReceivableSpecifiedTradeAccountingAccount *struct {
			ID       string  `xml:"ID"`
			TypeCode *string `xml:"TypeCode"`
		} `xml:"ReceivableSpecifiedTradeAccountingAccount"`
	} `xml:"SpecifiedLineTradeSettlement"`
}
type SpecifiedTradeAllowanceCharge struct {
	ChargeIndicator struct {
		Indicator bool `xml:"Indicator"`
	} `xml:"ChargeIndicator"`
	CalculationPercent *string `xml:"CalculationPercent"`
	ActualAmount       string  `xml:"ActualAmount"`
	ReasonCode         *string `xml:"ReasonCode"`
	Reason             *string `xml:"Reason"`
}

type ApplicableHeaderTradeAgreement struct {
	BuyerReference                    *string     `xml:"BuyerReference"`
	SellerTradeParty                  TradeParty  `xml:"SellerTradeParty"`
	SellerTaxRepresentativeTradeParty *TradeParty `xml:"SellerTaxRepresentativeTradeParty"`
	BuyerTradeParty                   TradeParty  `xml:"BuyerTradeParty"`
	AdditionalReferencedDocument      []*struct {
		IssuerAssignedID       string          `xml:"IssuerAssignedID"`
		TypeCode               string          `xml:"TypeCode"`
		Name                   *string         `xml:"Name"`
		ReferenceTypeCode      *string         `xml:"ReferenceTypeCode"`
		FormattedIssueDateTime *DateTimeFormat `xml:"FormattedIssueDateTime"`
		URIID                  *string         `xml:"URIID"`
		AttachmentBinaryObject *struct {
			Value    string `xml:",chardata"`
			MimeCode string `xml:"MimeCode,attr"`
			Filename string `xml:"Filename,attr"`
		} `xml:"AttachmentBinaryObject"`
	} `xml:"AdditionalReferencedDocument"`
}

type TradeParty struct {
	ID       string `xml:"ID,omitempty"`
	GlobalID *struct {
		Value    string `xml:",chardata"`
		SchemeID string `xml:"schemeID,attr"`
	} `xml:"GlobalID"`
	Name                string `xml:"Name"`
	DefinedTradeContact *struct {
		PersonName                      *string `xml:"PersonName"`
		TelephoneUniversalCommunication *struct {
			CompleteNumber string `xml:"CompleteNumber"`
		} `xml:"TelephoneUniversalCommunication"`
		EmailURIUniversalCommunication *struct {
			URIID string `xml:"URIID"`
		} `xml:"EmailURIUniversalCommunication"`
	} `xml:"DefinedTradeContact,omitempty"`
	PostalTradeAddress        *PostalTradeAddress `xml:"PostalTradeAddress"`
	URIUniversalCommunication struct {
		URIID struct {
			Value    string `xml:",chardata"`
			SchemeID string `xml:"schemeID,attr"`
		} `xml:"URIID"`
	} `xml:"URIUniversalCommunication"`
	SpecifiedTaxRegistration *[]struct {
		ID *struct {
			Value string `xml:",chardata"`
			//VA used for VAT-ID used in B2B, FC for tax number.
			SchemeID string `xml:"schemeID,attr"`
		} `xml:"ID"`
	} `xml:"SpecifiedTaxRegistration,omitempty"`
}

type ApplicableHeaderTradeDelivery struct {
	ShipToTradeParty               *TradeParty `xml:"ShipToTradeParty"`
	ActualDeliverySupplyChainEvent *struct {
		OccurrenceDateTime *DateTimeFormat `xml:"OccurrenceDateTime"`
	} `xml:"ActualDeliverySupplyChainEvent"`
	DespatchAdviceReferencedDocument *struct {
		IssuerAssignedID       string          `xml:"IssuerAssignedID"`
		FormattedIssueDateTime *DateTimeFormat `xml:"FormattedIssueDateTime"`
	} `xml:"DespatchAdviceReferencedDocument"`
	ReceivingAdviceReferencedDocument *struct {
		IssuerAssignedID       string          `xml:"IssuerAssignedID"`
		FormattedIssueDateTime *DateTimeFormat `xml:"FormattedIssueDateTime"`
	} `xml:"ReceivingAdviceReferencedDocument"`
	DeliveryNoteReferencedDocument *struct {
		IssuerAssignedID       string          `xml:"IssuerAssignedID"`
		FormattedIssueDateTime *DateTimeFormat `xml:"FormattedIssueDateTime"`
	} `xml:"DeliveryNoteReferencedDocument"`
}

type ApplicableHeaderTradeSettlement struct {
	InvoiceCurrencyCode                  string `xml:"InvoiceCurrencyCode"`
	SpecifiedTradeSettlementPaymentMeans []struct {
		TypeCode                               string  `xml:"TypeCode"`
		Information                            *string `xml:"Information"`
		ApplicableTradeSettlementFinancialCard *struct {
			ID             string `xml:"ID"`
			CardholderName string `xml:"CardholderName"`
		} `xml:"ApplicableTradeSettlementFinancialCard"`
		PayeePartyCreditorFinancialAccount *struct {
			IBANID      string `xml:"IBANID"`
			AccountName string `xml:"AccountName"`
		} `xml:"PayeePartyCreditorFinancialAccount"`
		PayerPartyDebtorFinancialAccount *struct {
			IBANID string `xml:"IBANID"`
		} `xml:"PayerPartyDebtorFinancialAccount"`
		PayeeSpecifiedCreditorFinancialInstitution *struct {
			BICID string `xml:"BICID"`
		} `xml:"PayeeSpecifiedCreditorFinancialInstitution"`
	} `xml:"SpecifiedTradeSettlementPaymentMeans"`
	ApplicableTradeTax []struct {
		CalculatedAmount      string `xml:"CalculatedAmount"`
		TypeCode              string `xml:"TypeCode"`
		BasisAmount           string `xml:"BasisAmount"`
		CategoryCode          string `xml:"CategoryCode"`
		RateApplicablePercent string `xml:"RateApplicablePercent"`
	} `xml:"ApplicableTradeTax"`
	SpecifiedTradeSettlementHeaderMonetarySummation struct {
		LineTotalAmount     float64 `xml:"LineTotalAmount"`
		TaxBasisTotalAmount string  `xml:"TaxBasisTotalAmount"`
		TaxTotalAmount      struct {
			Value      string `xml:",chardata"`
			CurrencyID string `xml:"currencyID,attr"`
		} `xml:"TaxTotalAmount"`
		TotalPrepaidAmount *string `xml:"TotalPrepaidAmount"`
		GrandTotalAmount   struct {
			Value      string `xml:",chardata"`
			CurrencyID string `xml:"currencyID,attr"`
		} `xml:"GrandTotalAmount"`
		DuePayableAmount string `xml:"DuePayableAmount"`
	} `xml:"SpecifiedTradeSettlementHeaderMonetarySummation"`
	PayeeTradeParty            *TradeParty `xml:"PayeeTradeParty"`
	SpecifiedTradePaymentTerms []struct {
		Description     *string `xml:"Description"`
		DueDateDateTime *struct {
			DateTimeString string `xml:"DateTimeString"`
			Format         string `xml:"format,attr"`
		} `xml:"DueDateDateTime"`
		PartialPaymentAmount               *string `xml:"PartialPaymentAmount"`
		ApplicableTradePaymentPenaltyTerms struct {
			BasisAmount string `xml:"BasisAmount"`
		} `xml:"ApplicableTradePaymentPenaltyTerms"`
		ApplicableTradePaymentDiscountTerms struct {
			BasisAmount string `xml:"BasisAmount"`
		} `xml:"ApplicableTradePaymentDiscountTerms"`
	} `xml:"SpecifiedTradePaymentTerms"`
	SpecifiedAdvancePayment []struct {
		PaidAmount                float64            `xml:"PaidAmount"`
		FormattedReceivedDateTime *DateTimeFormat    `xml:"FormattedReceivedDateTime"`
		IncludedTradeTax          []IncludedTradeTax `xml:"IncludedTradeTax"`
	} `xml:"SpecifiedAdvancePayment"`
	SpecifiedTradeAllowanceCharge []*struct {
		ChargeIndicator struct {
			Indicator bool `xml:"Indicator"`
		} `xml:"ChargeIndicator"`
		BasisAmount        *string `xml:"BasisAmount"`
		CalculationPercent *string `xml:"CalculationPercent"`
		ActualAmount       string  `xml:"ActualAmount"`
		ReasonCode         *string `xml:"ReasonCode"`
		Reason             *string `xml:"Reason"`
		CategoryTradeTax   struct {
			TypeCode              string  `xml:"TypeCode"`
			CategoryCode          string  `xml:"CategoryCode"`
			RateApplicablePercent *string `xml:"RateApplicablePercent"`
		} `xml:"CategoryTradeTax"`
	} `xml:"SpecifiedTradeAllowanceCharge"`
	InvoiceReferencedDcument []struct {
		IssuerAssignedID       string `xml:"IssuerAssignedID"`
		FormattedIssueDateTime *struct {
			DateTimeString *struct {
				Value  string `xml:",chardata"`
				Format string `xml:"format,attr"`
			} `xml:"DateTimeString"`
		} `xml:"FormattedIssueDateTime"`
	} `xml:"InvoiceReferencedDocument"`
	BillingSpecifiedPeriod *struct {
		StartDateTime *DateTimeFormat `xml:"StartDateTime"`
		EndDateTime   *DateTimeFormat `xml:"EndDateTime"`
		Description   *string         `xml:"Description"`
	} `xml:"BillingSpecifiedPeriod"`
}

type DateTimeFormat struct {
	DateTimeString string `xml:"DateTimeString"`
	Format         string `xml:"format,attr"`
}

type IncludedTradeTax struct {
	CalculatedAmount      string `xml:"CalculatedAmount"`
	TypeCode              string `xml:"TypeCode"`
	BasisAmount           string `xml:"BasisAmount"`
	CategoryCode          string `xml:"CategoryCode"`
	RateApplicablePercent string `xml:"RateApplicablePercent"`
}

type IncludedNote struct {
	ContentCode string `xml:"ContentCode"`
	Content     string `xml:"Content"`
	SubjectCode string `xml:"SubjectCode"`
}

type PostalTradeAddress struct {
	PostcodeCode           *string `xml:"PostcodeCode,omitempty"`
	LineOne                *string `xml:"LineOne,omitempty"`
	LineTwo                *string `xml:"LineTwo,omitempty"`
	LineThree              *string `xml:"LineThree,omitempty"`
	CityName               *string `xml:"CityName,omitempty"`
	CountryID              string  `xml:"CountryID"`
	CountrySubDivisionName *string `xml:"CountrySubDivisionName,omitempty"`
}
