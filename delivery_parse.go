package cii

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func goblNewDeliveryDetails(del *Delivery) (*bill.DeliveryDetails, error) {
	d := new(bill.DeliveryDetails)

	if del.Receiver != nil {
		d.Receiver = goblNewDeliveryParty(del.Receiver)

		// BT-71: Delivery location identifier stored on delivery identities
		// (not party identities) to match UBL mapping pattern.
		// GlobalID carries the scheme ID (BT-71-1), plain ID does not.
		if gid := firstPartyID(del.Receiver.GlobalID); gid != nil {
			id := &org.Identity{
				Code: cbc.Code(gid.Value),
			}
			if gid.SchemeID != "" {
				id.Ext = tax.ExtensionsOf(cbc.CodeMap{
					iso.ExtKeySchemeID: cbc.Code(gid.SchemeID),
				})
			}
			d.Identities = []*org.Identity{id}
		} else if pid := firstPartyID(del.Receiver.ID); pid != nil {
			d.Identities = []*org.Identity{
				{
					Code: cbc.Code(pid.Value),
				},
			}
		}
	}

	if del.Event != nil && del.Event.OccurrenceDate != nil && del.Event.OccurrenceDate.DateFormat != nil {
		deliveryDate, err := parseDate(del.Event.OccurrenceDate.DateFormat.Value)
		if err != nil {
			return nil, err
		}
		d.Date = &deliveryDate
	}

	if d.Receiver != nil || d.Date != nil || len(d.Identities) > 0 {
		return d, nil
	}
	return nil, nil
}

// goblNewDeliveryParty creates a GOBL party with only the BTs available
// for the delivery party (BT-70 name, BG-15 address).
func goblNewDeliveryParty(party *Party) *org.Party {
	p := &org.Party{
		Name: cleanString(party.Name),
	}
	if party.PostalTradeAddress != nil {
		p.Addresses = []*org.Address{
			goblNewAddress(party.PostalTradeAddress),
		}
	}
	if p.Name == "" && len(p.Addresses) == 0 {
		return nil
	}
	return p
}
