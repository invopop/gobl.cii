package cii

// CDARExchangedContext defines the structure of the ExchangedDocumentContext of the CDAR standard
type CDARExchangedContext struct {
	BusinessProcessParameter *CDARDocumentContextParameter `xml:"ram:BusinessProcessSpecifiedDocumentContextParameter,omitempty"`
	GuidelineParameter       *CDARDocumentContextParameter `xml:"ram:GuidelineSpecifiedDocumentContextParameter"`
}

// CDARDocumentContextParameter defines the structure of DocumentContextParameter of the CDAR standard
type CDARDocumentContextParameter struct {
	ID string `xml:"ram:ID"`
}

// CDARExchangedDocument defines the structure of the ExchangedDocument of the CDAR standard
type CDARExchangedDocument struct {
	ID                      string                  `xml:"ram:ID,omitempty"`
	Name                    string                  `xml:"ram:Name,omitempty"`
	TypeCode                string                  `xml:"ram:TypeCode,omitempty"`
	StatusCode              string                  `xml:"ram:StatusCode,omitempty"`
	IssueDateTime           *CDARIssueDateTime      `xml:"ram:IssueDateTime,omitempty"`
	LanguageID              string                  `xml:"ram:LanguageID,omitempty"`
	ElectronicPresentation  *CDARIndicator          `xml:"ram:ElectronicPresentationIndicator,omitempty"`
	VersionID               string                  `xml:"ram:VersionID,omitempty"`
	GlobalID                string                  `xml:"ram:GlobalID,omitempty"`
	IncludedNotes           []*CDARNote             `xml:"ram:IncludedNote,omitempty"`
	EffectivePeriod         *CDARSpecifiedPeriod    `xml:"ram:EffectiveSpecifiedPeriod,omitempty"`
	SenderTradeParty        *CDARTradeParty         `xml:"ram:SenderTradeParty,omitempty"`
	IssuerTradeParty        *CDARTradeParty         `xml:"ram:IssuerTradeParty,omitempty"`
	RecipientTradeParties   []*CDARTradeParty       `xml:"ram:RecipientTradeParty,omitempty"`
}

// CDARIssueDateTime defines the structure of IssueDateTime of the CDAR standard
type CDARIssueDateTime struct {
	DateTimeString *CDARDateTimeString `xml:"udt:DateTimeString"`
}

// CDARDateTimeString defines a date-time with format attribute
type CDARDateTimeString struct {
	Value  string `xml:",chardata"`
	Format string `xml:"format,attr,omitempty"`
}

// CDARQDTDateTimeString defines a date-time in qdt namespace with format attribute
type CDARQDTDateTimeString struct {
	Value  string `xml:",chardata"`
	Format string `xml:"format,attr,omitempty"`
}

// CDARFormattedIssueDateTime defines formatted issue date time with qdt namespace
type CDARFormattedIssueDateTime struct {
	DateTimeString *CDARQDTDateTimeString `xml:"qdt:DateTimeString"`
}

// CDARIndicator represents a boolean indicator
type CDARIndicator struct {
	Value bool `xml:"udt:Indicator"`
}

// CDARNote defines note in the RAM structure
type CDARNote struct {
	ContentCode string   `xml:"ram:ContentCode,omitempty"`
	Content     []string `xml:"ram:Content,omitempty"`
	SubjectCode string   `xml:"ram:SubjectCode,omitempty"`
}

// CDARSpecifiedPeriod defines a period with start and end dates
type CDARSpecifiedPeriod struct {
	StartDateTime *CDARIssueDateTime `xml:"ram:StartDateTime,omitempty"`
	EndDateTime   *CDARIssueDateTime `xml:"ram:EndDateTime,omitempty"`
}
