package cii

import (
	"slices"

	"github.com/invopop/gobl/addons/de/zugferd"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
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
		// BT-71: Delivery location identifier
		if len(inv.Delivery.Identities) > 0 {
			id := inv.Delivery.Identities[0]
			if d.Receiver == nil {
				d.Receiver = new(Party)
			}
			if id.Label != "" {
				// When scheme ID is present, use GlobalID
				d.Receiver.GlobalID = &PartyID{
					Value:    id.Code.String(),
					SchemeID: id.Label,
				}
			} else {
				d.Receiver.ID = &PartyID{
					Value: id.Code.String(),
				}
			}
		}
	} else if documentType := inv.Tax.Ext.Get(untdid.ExtKeyDocumentType); slices.Contains(inv.GetAddons(), zugferd.V2) && documentType.String() != "386" {
		// Helper for Zugferd BR-FX-EN-04 rule in case delivery
		// is not specified in the invoice (imported invoice)
		customerParty := inv.Customer
		if customerParty != nil && len(customerParty.Addresses) > 0 {
			d.Receiver = newDeliveryParty(customerParty)
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

// newDeliveryParty creates a Party with only the BTs available for
// the delivery party (BT-70 name, BG-15 address).
func newDeliveryParty(party *org.Party) *Party {
	if party == nil {
		return nil
	}
	return &Party{
		Name:               party.Name,
		PostalTradeAddress: newPostalTradeAddress(party.Addresses),
	}
}
