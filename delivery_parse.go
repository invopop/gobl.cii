package cii

import (
	"github.com/invopop/gobl/bill"
)

func goblNewDeliveryDetails(del *Delivery) (*bill.DeliveryDetails, error) {
	d := new(bill.DeliveryDetails)

	if del.Receiver != nil {
		d.Receiver = goblNewParty(del.Receiver)
	}

	if del.Event != nil && del.Event.OccurrenceDate != nil && del.Event.OccurrenceDate.DateFormat != nil {
		deliveryDate, err := parseDate(del.Event.OccurrenceDate.DateFormat.Value)
		if err != nil {
			return nil, err
		}
		d.Date = &deliveryDate
	}

	if d.Receiver != nil || d.Date != nil {
		return d, nil
	}
	return nil, nil
}
