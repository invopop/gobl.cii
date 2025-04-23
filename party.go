package cii

import (
	"fmt"

	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/fr"
	"github.com/invopop/gobl/tax"
)

// Party defines the structure of the TradePartyType of the CII standard
type Party struct {
	ID                        *PartyID                    `xml:"ram:ID,omitempty"`
	GlobalID                  *PartyID                    `xml:"ram:GlobalID,omitempty"`
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
	SchemeID string `xml:"schemeID,attr,omitempty"`
	Value    string `xml:",chardata"`
}

// PostalTradeAddress defines the structure of the PostalTradeAddress of the CII standard
type PostalTradeAddress struct {
	Postcode  string `xml:"ram:PostcodeCode,omitempty"`
	LineOne   string `xml:"ram:LineOne,omitempty"`
	LineTwo   string `xml:"ram:LineTwo,omitempty"`
	City      string `xml:"ram:CityName,omitempty"`
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
	ID   *PartyID `xml:"ram:ID"`
	Name string   `xml:"ram:TradingBusinessName"`
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

// newParty creates the SellerTradeParty part of a EN 16931 compliant invoice
func newParty(party *org.Party) *Party {
	if party == nil {
		return nil
	}
	p := &Party{
		Name:               party.Name,
		Contact:            newContact(party),
		PostalTradeAddress: newPostalTradeAddress(party.Addresses),
	}
	if party.TaxID != nil {
		// Assumes VAT ID being used instead of possible tax number
		p.SpecifiedTaxRegistration = []*SpecifiedTaxRegistration{
			{
				ID: &PartyID{
					Value:    party.TaxID.String(),
					SchemeID: mapGOBLTaxIDScheme(party.TaxID),
				},
			},
		}
	}
	if len(party.Identities) > 0 {
		for _, id := range party.Identities {

			if id.Ext.Has(iso.ExtKeySchemeID) {
				p.GlobalID = &PartyID{
					SchemeID: id.Ext[iso.ExtKeySchemeID].String(),
					Value:    id.Code.String(),
				}
			}

			// Hardcoded for chorus-pro.
			// As SIREN and SIRET are allways 0002, we can add the schemeID this way and not force the user to add the extension.
			if id.Type == fr.IdentityTypeSIREN || id.Type == fr.IdentityTypeSIRET {
				p.LegalOrganization = &LegalOrganization{
					ID: &PartyID{
						SchemeID: "0002",
						Value:    id.Code.String(),
					},
					Name: party.Alias,
				}
			}
		}
	}
	if len(party.Inboxes) > 0 {
		ib := party.Inboxes[0]
		if ib.Email != "" {
			p.URIUniversalCommunication = &URIUniversalCommunication{
				ID: &PartyID{
					Value:    ib.Email,
					SchemeID: SchemeIDEmail,
				},
			}
		} else {
			p.URIUniversalCommunication = &URIUniversalCommunication{
				ID: &PartyID{
					Value:    ib.Code.String(),
					SchemeID: ib.Scheme.String(),
				},
			}
		}
	}
	return p
}

func mapGOBLTaxIDScheme(id *tax.Identity) string {
	s := id.GetScheme()
	switch s {
	case tax.CategoryVAT:
		return "VA"
	default:
		// TODO: cover more versions here.
		return s.String()
	}
}

func newContact(p *org.Party) *Contact {
	c := new(Contact)
	if len(p.People) > 0 {
		pp := p.People[0]
		c.PersonName = contactName(pp.Name)
		if len(pp.Telephones) > 0 {
			c.Phone = &PhoneNumber{
				CompleteNumber: pp.Telephones[0].Number,
			}
		}
		if len(pp.Emails) > 0 {
			c.Email = &Email{
				URIID: pp.Emails[0].Address,
			}
		}
	}
	if c.Phone == nil && len(p.Telephones) > 0 {
		c.Phone = &PhoneNumber{
			CompleteNumber: p.Telephones[0].Number,
		}
	}
	// Fallback to adding the base company email addresses
	if c.Email == nil && len(p.Emails) > 0 {
		c.Email = &Email{
			URIID: p.Emails[0].Address,
		}
	}
	if c.PersonName == "" && c.Email == nil && c.Phone == nil {
		return nil
	}
	return c
}

func contactName(p *org.Name) string {
	g := p.Given
	s := p.Surname
	if g == "" && s == "" {
		return ""
	}
	if g == "" {
		return s
	}
	if s == "" {
		return g
	}
	return fmt.Sprintf("%s %s", g, s)
}
