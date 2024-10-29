package ctog

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) getParty(party *TradeParty) *org.Party {
	p := &org.Party{
		Name: party.Name,
	}

	if party.DefinedTradeContact != nil && party.DefinedTradeContact.PersonName != nil {
		p.People = []*org.Person{
			{
				Name: &org.Name{
					Given: *party.DefinedTradeContact.PersonName,
				},
			},
		}
	}

	if party.PostalTradeAddress != nil {
		p.Addresses = []*org.Address{
			parseAddress(party.PostalTradeAddress),
		}
	}

	if party.DefinedTradeContact != nil && party.DefinedTradeContact.TelephoneUniversalCommunication != nil {
		p.Telephones = []*org.Telephone{
			{
				Number: party.DefinedTradeContact.TelephoneUniversalCommunication.CompleteNumber,
			},
		}
	}

	if party.DefinedTradeContact != nil && party.DefinedTradeContact.EmailURIUniversalCommunication != nil {
		p.Emails = []*org.Email{
			{
				Address: party.DefinedTradeContact.EmailURIUniversalCommunication.URIID,
			},
		}
	}

	if party.SpecifiedTaxRegistration != nil {
		for _, taxReg := range *party.SpecifiedTaxRegistration {
			if taxReg.ID != nil {
				switch taxReg.ID.SchemeID {
				//Source (XML Document) https://ec.europa.eu/digital-building-blocks/sites/download/attachments/467108974/EN16931%20code%20lists%20values%20v13%20-%20used%20from%202024-05-15.xlsx?version=2&modificationDate=1712937109681&api=v2
				case "VA":
					p.TaxID = &tax.Identity{
						Country: l10n.TaxCountryCode(party.PostalTradeAddress.CountryID),
						Code:    cbc.Code(taxReg.ID.Value),
					}
				case "FC":
					identity := &org.Identity{
						Country: l10n.ISOCountryCode(party.PostalTradeAddress.CountryID),
						Code:    cbc.Code(taxReg.ID.Value),
					}
					if p.Identities == nil {
						p.Identities = make([]*org.Identity, 0)
					}
					p.Identities = append(p.Identities, identity)
				}
			}
		}
	}
	// Global ID is not yet mapped to the ISO 6523 ICD, its identifier is used as the label
	if party.GlobalID != nil {
		if p.Identities == nil {
			p.Identities = make([]*org.Identity, 0)
		}
		p.Identities = append(p.Identities, &org.Identity{
			Label: party.GlobalID.SchemeID,
			Code:  cbc.Code(party.GlobalID.Value),
		})
	}

	return p
}

func parseAddress(address *PostalTradeAddress) *org.Address {
	if address == nil {
		return nil
	}

	addr := &org.Address{
		Country: l10n.ISOCountryCode(address.CountryID),
	}

	if address.LineOne != nil {
		addr.Street = *address.LineOne
	}

	if address.LineTwo != nil {
		addr.StreetExtra = *address.LineTwo
	}

	if address.CityName != nil {
		addr.Locality = *address.CityName
	}

	if address.PostcodeCode != nil {
		addr.Code = *address.PostcodeCode
	}

	if address.CountrySubDivisionName != nil {
		addr.Region = *address.CountrySubDivisionName
	}

	return addr
}
