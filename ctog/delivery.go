package ctog

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func (c *Conversor) getDelivery(doc *Document) error {
	delivery := &bill.Delivery{}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ShipToTradeParty != nil {
		delivery.Receiver = c.getParty(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ShipToTradeParty)
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ActualDeliverySupplyChainEvent != nil &&
		doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ActualDeliverySupplyChainEvent.OccurrenceDateTime != nil {
		deliveryDate, err := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ActualDeliverySupplyChainEvent.OccurrenceDateTime.DateTimeString)
		if err != nil {
			return err
		}
		delivery.Date = &deliveryDate
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DeliveryNoteReferencedDocument != nil {
		delivery.Identities = []*org.Identity{
			{
				Code: cbc.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DeliveryNoteReferencedDocument.IssuerAssignedID),
			},
		}
	}

	if delivery.Receiver != nil || delivery.Date != nil || delivery.Identities != nil {
		c.inv.Delivery = delivery
	}
	return nil
}
