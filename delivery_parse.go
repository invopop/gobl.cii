package cii

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func goblNewDeliveryDetails(del *Delivery) (*bill.DeliveryDetails, error) {
	d := new(bill.DeliveryDetails)

	if del.Receiver != nil {
		d.Receiver = goblNewParty(del.Receiver)

		// BT-71: Delivery location identifier
		if del.Receiver.ID != nil && del.Receiver.ID.Value != "" {
			id := &org.Identity{
				Code: cbc.Code(del.Receiver.ID.Value),
			}
			if del.Receiver.ID.SchemeID != "" {
				id.Label = del.Receiver.ID.SchemeID
			}
			d.Identities = []*org.Identity{id}
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
