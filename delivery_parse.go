package cii

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func goblNewDeliveryDetails(del *Delivery) (*bill.DeliveryDetails, error) {
	d := new(bill.DeliveryDetails)

	if del.Receiver != nil {
		d.Receiver = goblNewDeliveryParty(del.Receiver)

		// BT-71: Delivery location identifier stored on delivery identities
		// (not party identities) to match UBL mapping pattern.
		// GlobalID carries the scheme ID, plain ID does not.
		if del.Receiver.GlobalID != nil && del.Receiver.GlobalID.Value != "" {
			d.Identities = []*org.Identity{
				{
					Code:  cbc.Code(del.Receiver.GlobalID.Value),
					Label: del.Receiver.GlobalID.SchemeID,
				},
			}
		} else if del.Receiver.ID != nil && del.Receiver.ID.Value != "" {
			d.Identities = []*org.Identity{
				{
					Code: cbc.Code(del.Receiver.ID.Value),
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
		Name: party.Name,
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
