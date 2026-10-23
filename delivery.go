package cii

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/org"
)

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
			d.Receiver = newDeliveryParty(inv.Delivery.Receiver)
		}
		if id := newDeliveryLocationID(inv.Delivery.Identities); id != nil {
			if d.Receiver == nil {
				d.Receiver = new(Party)
			}
			if id.SchemeID != "" {
				d.Receiver.GlobalID = []*PartyID{id}
			} else {
				d.Receiver.ID = []*PartyID{id}
			}
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

// NewDeliveryParty creates a Party with only the BTs available for
// the delivery party (BT-70 name, BG-15 address).
func NewDeliveryParty(party *org.Party) *Party {
	return newDeliveryParty(party)
}

func newDeliveryParty(party *org.Party) *Party {
	if party == nil {
		return nil
	}
	return &Party{
		Name:               party.Name,
		PostalTradeAddress: newPostalTradeAddress(party.Addresses),
	}
}

// newDeliveryLocationID builds the delivery location ID (BT-71) and its scheme
// (BT-71-1) from the first delivery identity, falling back to the label for
// documents imported before the scheme moved to the ISO scheme ID extension.
func newDeliveryLocationID(ids []*org.Identity) *PartyID {
	for _, id := range ids {
		if id == nil {
			continue
		}
		out := &PartyID{
			Value:    id.Code.String(),
			SchemeID: id.Ext.Get(iso.ExtKeySchemeID).String(),
		}
		if out.SchemeID == "" {
			out.SchemeID = id.Label
		}
		return out
	}
	return nil
}
