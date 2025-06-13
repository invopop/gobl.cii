package cii

import (
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
)

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

func (out *Invoice) addLines(lines []*bill.Line) error {
	var Lines []*Line

	for _, l := range lines {
		Lines = append(Lines, newLine(l))
	}

	out.Transaction.Lines = Lines
	return nil
}

func newLine(l *bill.Line) *Line {
	if l.Item == nil {
		return nil
	}
	it := l.Item

	lineItem := &Line{
		LineDoc: &LineDoc{
			ID: strconv.Itoa(l.Index),
		},
		Product: &Product{
			Name: it.Name,
		},
		Agreement: &LineAgreement{
			NetPrice: &NetPrice{
				// Amount: it.Price.Rescale(2).String(),
				Amount: it.Price.String(),
			},
		},
		Quantity: &LineDelivery{
			Quantity: &Quantity{
				Amount:   l.Quantity.String(),
				UnitCode: string(it.Unit.UNECE()),
			},
		},
		TradeSettlement: newTradeSettlement(l),
	}

	if it.Description != "" {
		lineItem.Product.Description = &it.Description
	}

	if len(l.Notes) > 0 {
		var notes []*Note
		for _, n := range l.Notes {
			notes = append(notes, &Note{
				SubjectCode: n.Key.String(),
				Content:     n.Text,
			})
		}
		lineItem.LineDoc.Note = notes
	}

	if len(it.Identities) > 0 {
		for _, id := range it.Identities {
			if id.Ext.Has(iso.ExtKeySchemeID) {
				lineItem.Product.GlobalID = &GlobalID{
					SchemeID: id.Ext[iso.ExtKeySchemeID].String(),
					Value:    id.Code.String(),
				}
			}
		}
	}

	return lineItem
}

func newTradeSettlement(l *bill.Line) *TradeSettlement {
	var taxes []*Tax
	for _, tax := range l.Taxes {
		t := makeTaxCategory(tax)
		taxes = append(taxes, t)
	}

	stlm := &TradeSettlement{
		ApplicableTradeTax: taxes,
		Sum: &Summation{
			Amount: l.Total.Rescale(2).String(),
		},
	}

	if len(l.Charges) > 0 || len(l.Discounts) > 0 {
		stlm.AllowanceCharge = newLineAllowanceCharges(l)
	}

	return stlm
}
