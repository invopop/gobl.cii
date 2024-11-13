package document

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
