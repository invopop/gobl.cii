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

	// BT-28/BT-45: Trading name (alias)
	if party.LegalOrganization != nil && party.LegalOrganization.Name != "" {
		if party.LegalOrganization.Name != p.Name {
			p.Alias = party.LegalOrganization.Name
		}
	}

	// BT-30/BT-47: Legal registration identifier
	if party.LegalOrganization != nil && party.LegalOrganization.ID != nil && party.LegalOrganization.ID.Value != "" {
		identity := &org.Identity{
			Code:  cbc.Code(party.LegalOrganization.ID.Value),
			Scope: org.IdentityScopeLegal,
		}
		if party.LegalOrganization.ID.SchemeID != "" {
			identity.Ext = tax.ExtensionsOf(cbc.CodeMap{
				iso.ExtKeySchemeID: cbc.Code(party.LegalOrganization.ID.SchemeID),
			})
		}
		p.Identities = append(p.Identities, identity)
	}

	// BT-29/BT-46: Seller/Buyer identifier
	if party.ID != nil && party.ID.Value != "" {
		identity := &org.Identity{
			Code: cbc.Code(party.ID.Value),
		}
		if party.ID.SchemeID != "" {
			identity.Ext = tax.ExtensionsOf(cbc.CodeMap{
				iso.ExtKeySchemeID: cbc.Code(party.ID.SchemeID),
			})
		}
		p.Identities = append(p.Identities, identity)
	}

	if party.PostalTradeAddress != nil {
		p.Addresses = []*org.Address{
			goblNewAddress(party.PostalTradeAddress),
		}
	}

	goblPartyContact(party, p)
	goblPartyTaxRegistrations(party, p)

	// Global ID is not yet mapped to the ISO 6523 ICD, its identifier is used as the label
	if party.GlobalID != nil {
		p.Identities = append(p.Identities, &org.Identity{
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				iso.ExtKeySchemeID: cbc.Code(party.GlobalID.SchemeID),
			}),
			Code: cbc.Code(party.GlobalID.Value),
		})
	}

	return p
}

func goblPartyContact(party *Party, p *org.Party) {
	if party.Contact != nil {
		if party.Contact.PersonName != "" {
			p.People = []*org.Person{
				{
					Name: &org.Name{
						Given: party.Contact.PersonName,
					},
				},
			}
		}
		if party.Contact.Phone != nil {
			p.Telephones = []*org.Telephone{
				{
					Number: party.Contact.Phone.CompleteNumber,
				},
			}
		}
		if party.Contact.Email != nil {
			p.Emails = []*org.Email{
				{
					Address: party.Contact.Email.URIID,
				},
			}
		}
	}
	if uc := party.URIUniversalCommunication; uc != nil && uc.ID != nil {
		if ep := goblEndpoint(uc.ID.SchemeID, uc.ID.Value); ep != nil {
			p.Endpoints = []*org.Endpoint{ep}
		}
	}
}

// goblEndpoint maps a CII URIUniversalCommunication (schemeID + value) to a
// GOBL endpoint URI:
//   - schemeID "EM" -> "mailto:<value>"
//   - any other EAS -> "iso6523-actorid-upis::<schemeID>:<value>"
func goblEndpoint(schemeID, value string) *org.Endpoint {
	if value == "" || schemeID == "" {
		return nil
	}
	var uri cbc.URI
	switch schemeID {
	case SchemeIDEmail: // email
		uri = cbc.URI(mailtoScheme + ":" + value)
	default:
		uri = cbc.URI(peppolEndpointScheme + "::" + schemeID + ":" + value)
	}
	return &org.Endpoint{URI: uri}
}

func goblPartyTaxRegistrations(party *Party, p *org.Party) {
	// Source: https://ec.europa.eu/digital-building-blocks/sites/download/attachments/467108974/EN16931%20code%20lists%20values%20v13%20-%20used%20from%202024-05-15.xlsx?version=2&modificationDate=1712937109681&api=v2
	for _, taxReg := range party.SpecifiedTaxRegistration {
		if taxReg.ID == nil || taxReg.ID.Value == "" {
			continue
		}
		switch taxReg.ID.SchemeID {
		case "VA":
			if identity, err := tax.ParseIdentity(taxReg.ID.Value); err == nil {
				if identity.Code != "" {
					p.TaxID = identity
				}
			} else {
				// Fallback to preserve the tax id
				p.TaxID = &tax.Identity{
					Country: l10n.TaxCountryCode(party.PostalTradeAddress.CountryID),
					Code:    cbc.Code(taxReg.ID.Value),
				}
			}
		case "FC":
			p.Identities = append(p.Identities, &org.Identity{
				Country: l10n.ISOCountryCode(party.PostalTradeAddress.CountryID),
				Code:    cbc.Code(taxReg.ID.Value),
			})
		}
	}
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
