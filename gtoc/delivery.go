package gtoc

import "github.com/invopop/gobl/bill"

// NewDelivery creates the ApplicableHeaderTradeDelivery part of a EN 16931 compliant invoice
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
		despatch := inv.Ordering.Despatch[0].Code.String()
		d.Despatch = &despatch
	}
	if inv.Ordering != nil && inv.Ordering.Receiving != nil {
		receiving := inv.Ordering.Receiving[0].Code.String()
		d.Receiving = &receiving
	}
	return d
}
