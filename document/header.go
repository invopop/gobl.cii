package document

// CII schema constants
const (
	RSM              = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	RAM              = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	QDT              = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
	UDT              = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
	BusinessProcess  = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
	GuidelineContext = "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0"
)

// ExchangedContext defines the structure of the ExchangedDocumentContext of the CII standard
type ExchangedContext struct {
	BusinessContext  *ExchangedContextParameter `xml:"ram:BusinessProcessSpecifiedDocumentContextParameter"`
	GuidelineContext *ExchangedContextParameter `xml:"ram:GuidelineSpecifiedDocumentContextParameter"`
}

// ExchangedContextParameter defines the structure of the ExchangedDocumentContextParameter of the CII standard
type ExchangedContextParameter struct {
	ID string `xml:"ram:ID"`
}

// Header a collection of data for a Cross Industry Invoice Header that is exchanged between two or more parties in written, printed or electronic form.
type Header struct {
	ID           string     `xml:"ram:ID"`
	TypeCode     string     `xml:"ram:TypeCode"`
	IssueDate    *IssueDate `xml:"ram:IssueDateTime"`
	IncludedNote []*Note    `xml:"ram:IncludedNote,omitempty"`
}

// IssueDate defines the structure of the IssueDateTime of the CII standard
type IssueDate struct {
	DateFormat *Date `xml:"udt:DateTimeString"`
}

// FormattedIssueDate defines the structure of the FormattedIssueDateTime of the CII standard
type FormattedIssueDate struct {
	DateFormat *Date `xml:"qdt:DateTimeString"`
}
