package gtoc

import "github.com/invopop/gobl/bill"

func NewDelivery(inv *bill.Invoice) *Delivery {
	d := &Delivery{}
	if inv.Delivery != nil {
		if inv.Delivery.Date != nil {
			d.Event = &Date{
				Date: formatIssueDate(*inv.Delivery.Date),
			}
		}
		if inv.Delivery.Receiver != nil {
			d.Receiver = NewBuyer(inv.Delivery.Receiver)
		}
	}
	if inv.Ordering != nil && inv.Ordering.Despatch != nil {
		d.Despatch = &Project{
			ID: (inv.Ordering.Despatch)[0].Code.String(),
		}
	}
	if inv.Ordering != nil && inv.Ordering.Receiving != nil {
		d.Receiving = &Project{
			ID: (inv.Ordering.Receiving)[0].Code.String(),
		}
	}
	return d
}
