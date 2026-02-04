package cii

// CDARAcknowledgement represents an AcknowledgementDocument in the CDAR standard
type CDARAcknowledgement struct {
	MultipleReferencesIndicator *CDARIndicator                `xml:"ram:MultipleReferencesIndicator,omitempty"`
	ID                          string                        `xml:"ram:ID,omitempty"`
	TypeCode                    string                        `xml:"ram:TypeCode,omitempty"`
	Name                        string                        `xml:"ram:Name,omitempty"`
	IssueDateTime               *CDARIssueDateTime            `xml:"ram:IssueDateTime,omitempty"`
	StatusCode                  string                        `xml:"ram:StatusCode,omitempty"`
	AcknowledgementStatusCode   string                        `xml:"ram:AcknowledgementStatusCode,omitempty"`
	ItemIdentificationID        string                        `xml:"ram:ItemIdentificationID,omitempty"`
	ReasonInformation           []string                      `xml:"ram:ReasonInformation,omitempty"`
	ChannelCode                 string                        `xml:"ram:ChannelCode,omitempty"`
	ProcessConditionCode        string                        `xml:"ram:ProcessConditionCode,omitempty"`
	ProcessCondition            []string                      `xml:"ram:ProcessCondition,omitempty"`
	Status                      []string                      `xml:"ram:Status,omitempty"`
	ReferenceReferencedDocument []*CDARReferencedDocument     `xml:"ram:ReferenceReferencedDocument"`
}

// CDARReferencedDocument represents a referenced document in the CDAR standard
type CDARReferencedDocument struct {
	IssuerAssignedID            string                          `xml:"ram:IssuerAssignedID,omitempty"`
	StatusCode                  string                          `xml:"ram:StatusCode,omitempty"`
	CopyIndicator               *CDARIndicator                  `xml:"ram:CopyIndicator,omitempty"`
	LineID                      string                          `xml:"ram:LineID,omitempty"`
	TypeCode                    string                          `xml:"ram:TypeCode,omitempty"`
	GlobalID                    string                          `xml:"ram:GlobalID,omitempty"`
	RevisionID                  string                          `xml:"ram:RevisionID,omitempty"`
	Name                        string                          `xml:"ram:Name,omitempty"`
	ReceiptDateTime             *CDARIssueDateTime              `xml:"ram:ReceiptDateTime,omitempty"`
	AttachmentBinaryObjects     []*CDARBinaryObject             `xml:"ram:AttachmentBinaryObject,omitempty"`
	ReferenceTypeCode           string                          `xml:"ram:ReferenceTypeCode,omitempty"`
	FormattedIssueDateTime      *CDARFormattedIssueDateTime     `xml:"ram:FormattedIssueDateTime,omitempty"`
	IssuerTradeParty            *CDARTradeParty                 `xml:"ram:IssuerTradeParty,omitempty"`
	RecipientTradeParties       []*CDARTradeParty               `xml:"ram:RecipientTradeParty,omitempty"`
	SpecifiedDocumentStatuses   []*CDARDocumentStatus           `xml:"ram:SpecifiedDocumentStatus,omitempty"`
}

// CDARBinaryObject represents a binary object attachment
type CDARBinaryObject struct {
	Value        string `xml:",chardata"`
	MimeCode     string `xml:"mimeCode,attr,omitempty"`
	Filename     string `xml:"filename,attr,omitempty"`
	CharacterSet string `xml:"characterSet,attr,omitempty"`
}

// CDARDocumentStatus represents a document status with conditions and reasons
type CDARDocumentStatus struct {
	ReferenceDateTime *CDARIssueDateTime `xml:"ram:ReferenceDateTime,omitempty"`
	ConditionCode     string             `xml:"ram:ConditionCode,omitempty"`
	ReasonCode        string             `xml:"ram:ReasonCode,omitempty"`
	Reason            []string           `xml:"ram:Reason,omitempty"`
	SequenceNumeric   int                `xml:"ram:SequenceNumeric,omitempty"`
}
