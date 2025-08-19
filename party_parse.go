package cii

import (
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func goblNewParty(party *Party) *org.Party {
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
			goblNewAddress(party.PostalTradeAddress),
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

	if uc := party.URIUniversalCommunication; uc != nil {
		if uc.ID.SchemeID == SchemeIDEmail {
			p.Inboxes = []*org.Inbox{
				{
					Email: uc.ID.Value,
				},
			}
		} else {
			p.Inboxes = []*org.Inbox{
				{
					Scheme: cbc.Code(uc.ID.SchemeID),
					Code:   cbc.Code(uc.ID.Value),
				},
			}
		}
	}

	if len(party.SpecifiedTaxRegistration) > 0 {
		for _, taxReg := range party.SpecifiedTaxRegistration {
			if taxReg.ID != nil && taxReg.ID.Value != "" {
				switch taxReg.ID.SchemeID {
				// Source (XML Document) https://ec.europa.eu/digital-building-blocks/sites/download/attachments/467108974/EN16931%20code%20lists%20values%20v13%20-%20used%20from%202024-05-15.xlsx?version=2&modificationDate=1712937109681&api=v2
				case "VA":
					// Parse the country code from the vat
					if identity, err := tax.ParseIdentity(taxReg.ID.Value); err == nil {
						p.TaxID = identity
					} else {
						// Fallback to preserve the tax id
						p.TaxID = &tax.Identity{
							Country: l10n.TaxCountryCode(party.PostalTradeAddress.CountryID),
							Code:    cbc.Code(taxReg.ID.Value),
						}
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

	// Handle LegalOrganization ID
	if party.LegalOrganization != nil && party.LegalOrganization.ID != nil && party.LegalOrganization.ID.Value != "" {
		if p.Identities == nil {
			p.Identities = make([]*org.Identity, 0)
		}
		identity := &org.Identity{
			Code: cbc.Code(party.LegalOrganization.ID.Value),
		}
		if party.PostalTradeAddress != nil {
			identity.Country = l10n.ISOCountryCode(party.PostalTradeAddress.CountryID)
		}
		if party.LegalOrganization.ID.SchemeID != "" {
			identity.Ext = tax.Extensions{
				iso.ExtKeySchemeID: cbc.Code(party.LegalOrganization.ID.SchemeID),
			}
		}
		p.Identities = append(p.Identities, identity)
	}

	// Global ID is not yet mapped to the ISO 6523 ICD, its identifier is used as the label
	if party.GlobalID != nil {
		if p.Identities == nil {
			p.Identities = make([]*org.Identity, 0)
		}
		p.Identities = append(p.Identities, &org.Identity{
			Ext: tax.Extensions{
				iso.ExtKeySchemeID: cbc.Code(party.GlobalID.SchemeID),
			},
			Code: cbc.Code(party.GlobalID.Value),
		})
	}

	return p
}

func goblNewAddress(address *PostalTradeAddress) *org.Address {
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
		addr.Code = cbc.Code(address.Postcode)
	}

	if address.Region != "" {
		addr.Region = address.Region
	}

	return addr
}
