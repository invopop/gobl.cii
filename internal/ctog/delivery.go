package ctog

import (
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
)

func (c *Converter) prepareDelivery(del *document.Delivery) error {
	d := &bill.DeliveryDetails{}

	if del.Receiver != nil {
		d.Receiver = c.getParty(del.Receiver)
	}

	if del.Event != nil && del.Event.OccurrenceDate != nil && del.Event.OccurrenceDate.DateFormat != nil {
		deliveryDate, err := ParseDate(del.Event.OccurrenceDate.DateFormat.Value)
		if err != nil {
			return err
		}
		d.Date = &deliveryDate
	}

	if d.Receiver != nil || d.Date != nil {
		c.inv.Delivery = d
	}
	return nil
}
