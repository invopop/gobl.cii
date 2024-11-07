package document

// Party defines the structure of the TradePartyType of the CII standard
type Party struct {
	ID                        *PartyID                    `xml:"ram:ID,omitempty"`
	GlobalID                  []*PartyID                  `xml:"ram:GlobalID,omitempty"`
	Name                      string                      `xml:"ram:Name,omitempty"`
	Description               string                      `xml:"ram:Description,omitempty"`
	LegalOrganization         *LegalOrganization          `xml:"ram:SpecifiedLegalOrganization,omitempty"`
	Contact                   *Contact                    `xml:"ram:DefinedTradeContact,omitempty"`
	PostalTradeAddress        *PostalTradeAddress         `xml:"ram:PostalTradeAddress,omitempty"`
	URIUniversalCommunication *URIUniversalCommunication  `xml:"ram:URIUniversalCommunication,omitempty"`
	SpecifiedTaxRegistration  []*SpecifiedTaxRegistration `xml:"ram:SpecifiedTaxRegistration,omitempty"`
}

// PartyID defines the structure of the ID of the CII standard
type PartyID struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
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
	ID *PartyID `xml:"ram:URIID"`
}

// SpecifiedTaxRegistration defines the structure of the SpecifiedTaxRegistration of the CII standard
type SpecifiedTaxRegistration struct {
	ID *PartyID `xml:"ram:ID"`
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
