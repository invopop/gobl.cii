package cii

import (
	"strings"

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
	for _, partyID := range party.ID {
		if partyID == nil || partyID.Value == "" {
			continue
		}
		identity := &org.Identity{
			Code: cbc.Code(partyID.Value),
		}
		if partyID.SchemeID != "" {
			identity.Ext = tax.ExtensionsOf(cbc.CodeMap{
				iso.ExtKeySchemeID: cbc.Code(partyID.SchemeID),
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
	for _, gid := range party.GlobalID {
		if gid == nil {
			continue
		}
		p.Identities = append(p.Identities, &org.Identity{
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				iso.ExtKeySchemeID: cbc.Code(gid.SchemeID),
			}),
			Code: cbc.Code(gid.Value),
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
	if uc := party.URIUniversalCommunication; uc != nil {
		if uc.ID.SchemeID == SchemeIDEmail {
			p.Inboxes = []*org.Inbox{{Email: uc.ID.Value}}
		} else {
			p.Inboxes = []*org.Inbox{{Scheme: cbc.Code(uc.ID.SchemeID), Code: cbc.Code(uc.ID.Value)}}
		}
	}
}

func goblPartyTaxRegistrations(party *Party, p *org.Party) {
	// Source: https://ec.europa.eu/digital-building-blocks/sites/download/attachments/467108974/EN16931%20code%20lists%20values%20v13%20-%20used%20from%202024-05-15.xlsx?version=2&modificationDate=1712937109681&api=v2
	country := partyCountryID(party)
	for _, taxReg := range party.SpecifiedTaxRegistration {
		if taxReg.ID == nil || taxReg.ID.Value == "" {
			continue
		}
		if isVATRegistration(taxReg.ID.SchemeID) {
			// BT-31/BT-48/BT-63: VAT identifier
			if identity, err := tax.ParseIdentity(taxReg.ID.Value); err == nil {
				if identity.Code != "" {
					p.TaxID = identity
				}
			} else {
				// Fallback to preserve the tax id
				p.TaxID = &tax.Identity{
					Country: l10n.TaxCountryCode(country),
					Code:    cbc.Code(taxReg.ID.Value),
				}
			}
			continue
		}
		// BT-32/BT-49: every other registration keeps the tax scope so it
		// converts back out as a SpecifiedTaxRegistration. The scheme code
		// itself is not modelled in GOBL: EN 16931 only ever uses "FC"
		// here, and the identity type is reserved for national identifier
		// types such as SIREN.
		p.Identities = append(p.Identities, &org.Identity{
			Scope:   org.IdentityScopeTax,
			Country: l10n.ISOCountryCode(country),
			Code:    cbc.Code(taxReg.ID.Value),
		})
	}
}

// isVATRegistration reports whether the scheme identifier of a
// SpecifiedTaxRegistration marks it as the party's VAT identifier. "VA" is
// the EN 16931 code; "VAT" is the UBL one, common enough in CII documents
// in the wild that dropping the VAT number over it would be worse than
// accepting it.
func isVATRegistration(schemeID string) bool {
	switch strings.ToUpper(schemeID) {
	case SchemeIDVAT, SchemeIDVATAlt:
		return true
	}
	return false
}

// partyCountryID returns the party's postal address country, or an empty
// string when the party has no address. BG-5 and BG-8 are mandatory in EN
// 16931, but parsing must not depend on the document being conformant.
func partyCountryID(party *Party) string {
	if party.PostalTradeAddress == nil {
		return ""
	}
	return party.PostalTradeAddress.CountryID
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

// firstPartyID returns the first identifier carrying a value, or nil when the
// slice holds none. BT-29/BT-46 are 0..n, but a few mappings (BT-71 delivery
// location) only ever deal with a single entry.
func firstPartyID(ids []*PartyID) *PartyID {
	for _, id := range ids {
		if id != nil && id.Value != "" {
			return id
		}
	}
	return nil
}
