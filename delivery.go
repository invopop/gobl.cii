package cii

import "github.com/invopop/gobl/bill"

// Delivery defines the structure of ApplicableHeaderTradeDelivery of the CII standard
type Delivery struct {
	Receiver  *Party      `xml:"ram:ShipToTradeParty,omitempty"`
	Event     *ChainEvent `xml:"ram:ActualDeliverySupplyChainEvent,omitempty"`
	Despatch  *IssuerID   `xml:"ram:DespatchAdviceReferencedDocument,omitempty"`
	Receiving *IssuerID   `xml:"ram:ReceivingAdviceReferencedDocument,omitempty"`
}

// ChainEvent defines the structure of the OccurrenceDateTime of the CII standard
type ChainEvent struct {
	OccurrenceDate *IssueDate `xml:"ram:OccurrenceDateTime"`
}

// prepareDelivery creates the ApplicableHeaderTradeDelivery part of a EN 16931 compliant invoice
func newDelivery(inv *bill.Invoice) *Delivery {
	d := new(Delivery)
	if inv.Delivery != nil {
		if inv.Delivery.Date != nil {
			d.Event = &ChainEvent{
				OccurrenceDate: &IssueDate{
					DateFormat: &Date{
						Value:  formatIssueDate(*inv.Delivery.Date),
						Format: issueDateFormat,
					},
				},
			}
		}
		if inv.Delivery.Receiver != nil {
			d.Receiver = newParty(inv.Delivery.Receiver)
		}
	}
	if inv.Ordering != nil && inv.Ordering.Despatch != nil {
		despatch := inv.Ordering.Despatch[0].Code.String()
		d.Despatch = &IssuerID{
			ID: despatch,
		}
	}
	if inv.Ordering != nil && inv.Ordering.Receiving != nil {
		receiving := inv.Ordering.Receiving[0].Code.String()
		d.Receiving = &IssuerID{
			ID: receiving,
		}
	}
	return d
}
