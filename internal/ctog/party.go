package ctog

import (
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) getParty(party *document.Party) *org.Party {
	p := &org.Party{
		Name: party.Name,
	}

	if party.Contact != nil && party.Contact.PersonName != "" {
		p.People = []*org.Person{
			{
				Name: &org.Name{
					Given: party.Contact.PersonName,
				},
			},
		}
	}

	if party.PostalTradeAddress != nil {
		p.Addresses = []*org.Address{
			parseAddress(party.PostalTradeAddress),
		}
	}

	if party.Contact != nil && party.Contact.Phone != nil {
		p.Telephones = []*org.Telephone{
			{
				Number: party.Contact.Phone.CompleteNumber,
			},
		}
	}

	if party.Contact != nil && party.Contact.Email != nil {
		p.Emails = []*org.Email{
			{
				Address: party.Contact.Email.URIID,
			},
		}
	}

	if len(party.SpecifiedTaxRegistration) > 0 {
		for _, taxReg := range party.SpecifiedTaxRegistration {
			if taxReg.ID != nil && taxReg.ID.Value != "" {
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
	if len(party.GlobalID) > 0 {
		for _, id := range party.GlobalID {
			if id.Value != "" {
				if p.Identities == nil {
					p.Identities = make([]*org.Identity, 0)
				}
				p.Identities = append(p.Identities, &org.Identity{
					Label: id.SchemeID,
					Code:  cbc.Code(id.Value),
				})
			}
		}
	}

	return p
}

func parseAddress(address *document.PostalTradeAddress) *org.Address {
	if address == nil {
		return nil
	}

	addr := &org.Address{
		Country: l10n.ISOCountryCode(address.CountryID),
	}

	if address.LineOne != "" {
		addr.Street = address.LineOne
	}

	if address.LineTwo != "" {
		addr.StreetExtra = address.LineTwo
	}

	if address.City != "" {
		addr.Locality = address.City
	}

	if address.Postcode != "" {
		addr.Code = address.Postcode
	}

	if address.Region != "" {
		addr.Region = address.Region
	}

	return addr
}
