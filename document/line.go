package document

// Line defines the structure of the IncludedSupplyChainTradeLineItem in the CII standard
type Line struct {
	LineDoc         *LineDoc         `xml:"ram:AssociatedDocumentLineDocument"`
	Product         *Product         `xml:"ram:SpecifiedTradeProduct"`
	Agreement       *LineAgreement   `xml:"ram:SpecifiedLineTradeAgreement"`
	Quantity        *LineDelivery    `xml:"ram:SpecifiedLineTradeDelivery"`
	TradeSettlement *TradeSettlement `xml:"ram:SpecifiedLineTradeSettlement"`
}

// LineDoc defines the structure of the AssociatedDocumentLineDocument in the CII standard
type LineDoc struct {
	ID   string  `xml:"ram:LineID"`
	Note []*Note `xml:"ram:IncludedNote,omitempty"`
}

// LineAgreement defines the structure of the SpecifiedLineTradeAgreement in the CII standard
type LineAgreement struct {
	NetPrice *NetPrice `xml:"ram:NetPriceProductTradePrice"`
}

// NetPrice defines the structure of the NetPriceProductTradePrice in the CII standard
type NetPrice struct {
	Amount string `xml:"ram:ChargeAmount"`
}

// LineDelivery defines the structure of the SpecifiedLineTradeDelivery in the CII standard
type LineDelivery struct {
	Quantity *Quantity `xml:"ram:BilledQuantity"`
}

// Product defines the structure of the SpecifiedTradeProduct of the CII standard
type Product struct {
	GlobalID         *GlobalID         `xml:"ram:GlobalID,omitempty"`
	SellerAssignedID *string           `xml:"ram:SellerAssignedID,omitempty"`
	BuyerAssignedID  *string           `xml:"ram:BuyerAssignedID,omitempty"`
	Name             string            `xml:"ram:Name"`
	Description      *string           `xml:"ram:Description,omitempty"`
	Characteristics  []*Characteristic `xml:"ram:ApplicableProductCharacteristic,omitempty"`
	Classification   *Classification   `xml:"ram:DesignatedProductClassification,omitempty"`
	Origin           *string           `xml:"ram:OriginTradeCountry>ram:ID,omitempty"`
}

// Classification defines the structure of the DesignatedProductClassification of the CII standard
type Classification struct {
	Code *ListID `xml:"ram:ClassCode,omitempty"`
}

// GlobalID defines the structure of the GlobalID of the CII standard
type GlobalID struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

// ListID defines the structure of the ListID of the CII standard
type ListID struct {
	ID     string `xml:"ram:ID,omitempty"`
	ListID string `xml:"ListID,attr,omitempty"`
}

// Characteristic defines the structure of the ApplicableProductCharacteristic of the CII standard
type Characteristic struct {
	Description string `xml:"ram:Description,omitempty"`
	Value       string `xml:"ram:Value,omitempty"`
}

// Quantity defines the structure of the quantity with its attributes for the CII standard
type Quantity struct {
	Amount   string `xml:",chardata"`
	UnitCode string `xml:"unitCode,attr"`
}

// TradeSettlement defines the structure of the SpecifiedLineTradeSettlement of the CII standard
type TradeSettlement struct {
	ApplicableTradeTax []*Tax             `xml:"ram:ApplicableTradeTax"`
	AllowanceCharge    []*AllowanceCharge `xml:"ram:SpecifiedTradeAllowanceCharge,omitempty"`
	Sum                *Summation         `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation"`
}

// Summation defines the structure of the SpecifiedTradeSettlementLineMonetarySummation of the CII standard
type Summation struct {
	Amount string `xml:"ram:LineTotalAmount"`
}
