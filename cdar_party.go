package cii

// CDARTradeParty represents party information (sender, issuer, recipient)
type CDARTradeParty struct {
	IDs                       []*CDARID                       `xml:"ram:ID,omitempty"`
	GlobalIDs                 []*CDARGlobalID                 `xml:"ram:GlobalID,omitempty"`
	Name                      string                          `xml:"ram:Name,omitempty"`
	RoleCode                  string                          `xml:"ram:RoleCode,omitempty"`
	DefinedTradeContacts      []*CDARTradeContact             `xml:"ram:DefinedTradeContact,omitempty"`
	PostalTradeAddress        *CDARTradeAddress               `xml:"ram:PostalTradeAddress,omitempty"`
	URIUniversalCommunication *CDARUniversalCommunication     `xml:"ram:URIUniversalCommunication,omitempty"`
}

// CDARID represents an identifier
type CDARID struct {
	Value    string `xml:",chardata"`
	SchemeID string `xml:"schemeID,attr,omitempty"`
}

// CDARGlobalID represents global identifier with scheme
type CDARGlobalID struct {
	Value    string `xml:",chardata"`
	SchemeID string `xml:"schemeID,attr,omitempty"`
}

// CDARTradeContact represents contact details
type CDARTradeContact struct {
	ID                              string                          `xml:"ram:ID,omitempty"`
	PersonName                      string                          `xml:"ram:PersonName,omitempty"`
	DepartmentName                  string                          `xml:"ram:DepartmentName,omitempty"`
	TypeCode                        string                          `xml:"ram:TypeCode,omitempty"`
	TelephoneUniversalCommunication []*CDARUniversalCommunication   `xml:"ram:TelephoneUniversalCommunication,omitempty"`
	FaxUniversalCommunication       []*CDARUniversalCommunication   `xml:"ram:FaxUniversalCommunication,omitempty"`
	EmailURIUniversalCommunication  *CDARUniversalCommunication     `xml:"ram:EmailURIUniversalCommunication,omitempty"`
}

// CDARTradeAddress represents address information
type CDARTradeAddress struct {
	PostcodeCode string `xml:"ram:PostcodeCode,omitempty"`
	LineOne      string `xml:"ram:LineOne,omitempty"`
	LineTwo      string `xml:"ram:LineTwo,omitempty"`
	LineThree    string `xml:"ram:LineThree,omitempty"`
	LineFour     string `xml:"ram:LineFour,omitempty"`
	LineFive     string `xml:"ram:LineFive,omitempty"`
	StreetName   string `xml:"ram:StreetName,omitempty"`
	CityName     string `xml:"ram:CityName,omitempty"`
	CountryID    string `xml:"ram:CountryID,omitempty"`
	CountryName  string `xml:"ram:CountryName,omitempty"`
}

// CDARUniversalCommunication represents electronic communication details
type CDARUniversalCommunication struct {
	URIID          *CDARURIID `xml:"ram:URIID,omitempty"`
	CompleteNumber string     `xml:"ram:CompleteNumber,omitempty"`
}

// CDARURIID represents electronic address identifier with scheme
type CDARURIID struct {
	Value    string `xml:",chardata"`
	SchemeID string `xml:"schemeID,attr,omitempty"`
}
