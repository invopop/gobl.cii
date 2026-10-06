package cii

import (
	"fmt"
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
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
	ID string `xml:"ram:LineID"`
	// Sub-invoice lines (CII extended profile, EXT-FR-FE-162/163): the line
	// this one belongs to and whether it is a GROUP, DETAIL or INFORMATION
	// line. LineStatusCode is only read.
	ParentLineID         string  `xml:"ram:ParentLineID,omitempty"`
	LineStatusCode       string  `xml:"ram:LineStatusCode,omitempty"`
	LineStatusReasonCode string  `xml:"ram:LineStatusReasonCode,omitempty"`
	Note                 []*Note `xml:"ram:IncludedNote,omitempty"`
}

// Sub-invoice line types (EXT-FR-FE-163, ram:LineStatusReasonCode). Only
// DETAIL lines and lines without a type count towards the totals: a GROUP
// line restates the sum of its DETAIL lines, and an INFORMATION line is
// purely descriptive.
const (
	lineStatusGroup       = "GROUP"
	lineStatusDetail      = "DETAIL"
	lineStatusInformation = "INFORMATION"
)

// LineAgreement defines the structure of the SpecifiedLineTradeAgreement in the CII standard
type LineAgreement struct {
	OrderReference      *LineOrderReference `xml:"ram:BuyerOrderReferencedDocument,omitempty"`
	AdditionalReference *LineDocReference   `xml:"ram:AdditionalReferencedDocument,omitempty"`
	NetPrice            *NetPrice           `xml:"ram:NetPriceProductTradePrice"`
	// The XSD sequence places the item seller after the prices, before
	// ItemBuyerTradeParty.
	ItemSellerParty *Party `xml:"ram:ItemSellerTradeParty,omitempty"`
}

// LineDocReference defines the structure of AdditionalReferencedDocument at line level
type LineDocReference struct {
	ID       string  `xml:"ram:IssuerAssignedID"`
	TypeCode string  `xml:"ram:TypeCode"`
	RefCode  *string `xml:"ram:ReferenceTypeCode,omitempty"`
}

// LineOrderReference defines the structure of BuyerOrderReferencedDocument at line level
type LineOrderReference struct {
	LineID string `xml:"ram:LineID,omitempty"`
}

// NetPrice defines the structure of the NetPriceProductTradePrice in the CII standard
type NetPrice struct {
	Amount       string    `xml:"ram:ChargeAmount"`
	BaseQuantity *Quantity `xml:"ram:BasisQuantity,omitempty"`
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
	Value  string `xml:",chardata"`
	ListID string `xml:"listID,attr,omitempty"`
}

// Characteristic defines the structure of the ApplicableProductCharacteristic
// of the CII standard. ValueMeasure is only ever read: documents from senders
// using the CII extended profile may carry one, but the EN 16931 profiles do
// not accept it on output.
type Characteristic struct {
	Description  string    `xml:"ram:Description,omitempty"`
	ValueMeasure *Quantity `xml:"ram:ValueMeasure,omitempty"`
	Value        string    `xml:"ram:Value,omitempty"`
}

// Quantity defines the structure of the quantity with its attributes for the CII standard
type Quantity struct {
	Amount   string `xml:",chardata"`
	UnitCode string `xml:"unitCode,attr"`
}

// TradeSettlement defines the structure of the SpecifiedLineTradeSettlement of the CII standard
type TradeSettlement struct {
	ApplicableTradeTax []*Tax             `xml:"ram:ApplicableTradeTax"`
	Period             *Period            `xml:"ram:BillingSpecifiedPeriod,omitempty"`
	AllowanceCharge    []*AllowanceCharge `xml:"ram:SpecifiedTradeAllowanceCharge,omitempty"`
	Sum                *Summation         `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation"`
	AccountingAccount  *AccountingAccount `xml:"ram:ReceivableSpecifiedTradeAccountingAccount,omitempty"`
}

// AccountingAccount defines the structure of ReceivableSpecifiedTradeAccountingAccount
type AccountingAccount struct {
	ID string `xml:"ram:ID,omitempty"`
}

// Summation defines the structure of the SpecifiedTradeSettlementLineMonetarySummation of the CII standard
type Summation struct {
	Amount string `xml:"ram:LineTotalAmount"`
}

func (out *Invoice) addLines(inv *bill.Invoice, ctx Context) error {
	var Lines []*Line

	for _, l := range inv.Lines {
		ccy := lineCurrency(inv, l)
		line := newLine(l, ccy, ctx)
		if line != nil && writesSubLines(ctx, l) {
			Lines = append(Lines, newGroupLines(l, line, ccy)...)
			continue
		}
		Lines = append(Lines, line)
	}

	out.Transaction.Lines = Lines
	return nil
}

// writesSubLines reports whether a line's breakdown is written out as
// sub-invoice lines. Only the extended profiles define them, and a line's own
// allowances or charges would be lost on a GROUP line, which no total counts;
// such a line is written as a single line, as every other profile does.
func writesSubLines(ctx Context, l *bill.Line) bool {
	if len(l.Breakdown) == 0 || len(l.Discounts) > 0 || len(l.Charges) > 0 {
		return false
	}
	return ctx.Is(ContextFacturXExtendedV1) || ctx.Is(ContextZUGFeRDExtendedV2) || isFranceExtended(&ctx)
}

// newGroupLines writes a line with a breakdown as sub-invoice lines. When a
// sub-line carries a price the line becomes a GROUP restating the sum of its
// DETAIL lines, with no tax of its own. Otherwise its price stands and the
// sub-lines only describe it. Sub-lines have no taxes, so each takes the
// line's, and unpriced ones are INFORMATION lines that count towards nothing.
func newGroupLines(l *bill.Line, group *Line, ccy string) []*Line {
	priced := false
	for _, sl := range l.Breakdown {
		if sl != nil && sl.Item != nil && sl.Item.Price != nil {
			priced = true
		}
	}

	lines := []*Line{group}
	sum := num.AmountZero
	for i, sl := range l.Breakdown {
		if sl == nil || sl.Item == nil {
			continue
		}
		child := newSubLine(sl, l, fmt.Sprintf("%s.%d", group.LineDoc.ID, i+1), group.LineDoc.ID, ccy)
		if child.LineDoc.LineStatusReasonCode == lineStatusDetail {
			amount, _ := num.AmountFromString(child.TradeSettlement.Sum.Amount)
			sum = sum.MatchPrecision(amount).Add(amount)
		}
		lines = append(lines, child)
	}

	if priced {
		group.LineDoc.LineStatusReasonCode = lineStatusGroup
		group.TradeSettlement.ApplicableTradeTax = nil
		// BR-FREXT-08: a GROUP line's amount is the sum of its DETAIL lines.
		group.TradeSettlement.Sum.Amount = rescaleToCurrency(sum, ccy)
	} else {
		group.LineDoc.LineStatusReasonCode = lineStatusDetail
	}
	return lines
}

// newSubLine writes a sub-line under its parent. A sub-line counts per unit of
// its parent, while a sub-invoice line states its full quantity and amounts,
// so they are multiplied by the parent's quantity.
func newSubLine(sl *bill.SubLine, parent *bill.Line, id, parentID, ccy string) *Line {
	it := sl.Item
	qty := parent.Quantity
	line := &Line{
		LineDoc: &LineDoc{
			ID:                   id,
			ParentLineID:         parentID,
			LineStatusReasonCode: lineStatusDetail,
		},
		Product:   newProduct(it),
		Agreement: &LineAgreement{},
		Quantity: &LineDelivery{
			Quantity: &Quantity{
				Amount:   sl.Quantity.Multiply(qty).String(),
				UnitCode: untdidUnit(it.Ext, it.Unit).String(),
			},
		},
		TradeSettlement: &TradeSettlement{},
	}
	for _, t := range parent.Taxes {
		line.TradeSettlement.ApplicableTradeTax = append(line.TradeSettlement.ApplicableTradeTax, makeTaxCategory(t))
	}

	if it.Price == nil || sl.Total == nil {
		// The extended schemas still want a price and an amount, so an
		// INFORMATION line states zero for both, as the official examples do.
		line.LineDoc.LineStatusReasonCode = lineStatusInformation
		line.Agreement.NetPrice = &NetPrice{Amount: rescaleToCurrency(num.AmountZero, ccy)}
		line.TradeSettlement.Sum = &Summation{Amount: rescaleToCurrency(num.AmountZero, ccy)}
	} else {
		line.Agreement.NetPrice = &NetPrice{Amount: it.Price.String()}
		line.TradeSettlement.Sum = &Summation{
			Amount: rescaleToCurrency(sl.Total.Multiply(qty), ccy),
		}
		line.TradeSettlement.AllowanceCharge = newLineAllowanceCharges(
			scaleLineCharges(sl.Charges, qty), scaleLineDiscounts(sl.Discounts, qty), ccy,
		)
	}

	line.LineDoc.Note = newLineNotes(sl.Notes)
	line.TradeSettlement.Period = newLinePeriod(sl.Period)
	line.Agreement.AdditionalReference = newLineDocReference(sl.Identifier)
	if sl.Order != "" {
		line.Agreement.OrderReference = &LineOrderReference{LineID: sl.Order.String()}
	}
	if sl.Cost != "" {
		line.TradeSettlement.AccountingAccount = &AccountingAccount{ID: sl.Cost.String()}
	}
	return line
}

// scaleLineCharges multiplies sub-line charges by the parent's quantity.
func scaleLineCharges(charges []*bill.LineCharge, qty num.Amount) []*bill.LineCharge {
	out := make([]*bill.LineCharge, 0, len(charges))
	for _, c := range charges {
		sc := *c
		sc.Amount = c.Amount.Multiply(qty)
		if c.Base != nil {
			b := c.Base.Multiply(qty)
			sc.Base = &b
		}
		out = append(out, &sc)
	}
	return out
}

// scaleLineDiscounts multiplies sub-line discounts by the parent's quantity.
func scaleLineDiscounts(discounts []*bill.LineDiscount, qty num.Amount) []*bill.LineDiscount {
	out := make([]*bill.LineDiscount, 0, len(discounts))
	for _, d := range discounts {
		sd := *d
		sd.Amount = d.Amount.Multiply(qty)
		if d.Base != nil {
			b := d.Base.Multiply(qty)
			sd.Base = &b
		}
		out = append(out, &sd)
	}
	return out
}

// newCharacteristics builds the BG-32 item attributes from the item's
// attributes: the label names the attribute (BT-160) and the value (BT-161)
// presents whichever value the attribute holds. Amounts carry their unit in
// the value text, as ram:ValueMeasure is rejected by the Factur-X and ZUGFeRD
// EN 16931 schemas and warned against by CII-SR-070 everywhere else.
func newCharacteristics(attrs []*org.Attribute) []*Characteristic {
	chars := make([]*Characteristic, 0, len(attrs))
	for _, attr := range attrs {
		c := &Characteristic{Description: characteristicName(attr)}
		switch {
		case attr.Amount != nil:
			c.Value = attr.Amount.String()
			if label := unitLabel(attr.Unit, untdidUnit(attr.Ext, attr.Unit)); label != "" {
				c.Value += " " + label
			}
		case attr.Text != "":
			c.Value = attr.Text
		case attr.Code != "":
			c.Value = attr.Code.String()
		case attr.Date != nil:
			c.Value = attr.Date.String()
		}
		if c.Description == "" || c.Value == "" {
			continue
		}
		chars = append(chars, c)
	}
	if len(chars) == 0 {
		return nil
	}
	return chars
}

// characteristicName names an attribute for BT-160, falling back to the key or
// type when it carries no label.
func characteristicName(attr *org.Attribute) string {
	switch {
	case attr.Label != "":
		return attr.Label
	case attr.Key != cbc.KeyEmpty:
		return attr.Key.String()
	default:
		return attr.Type.String()
	}
}

func newLine(l *bill.Line, ccy string, ctx Context) *Line {
	if l.Item == nil {
		return nil
	}
	it := l.Item

	lineItem := &Line{
		LineDoc: &LineDoc{
			ID:   strconv.Itoa(l.Index),
			Note: newLineNotes(l.Notes),
		},
		Product: newProduct(it),
		Agreement: &LineAgreement{
			NetPrice: &NetPrice{
				Amount: it.Price.String(),
			},
			AdditionalReference: newLineDocReference(l.Identifier),
		},
		Quantity: &LineDelivery{
			Quantity: &Quantity{
				Amount:   l.Quantity.String(),
				UnitCode: untdidUnit(it.Ext, it.Unit).String(),
			},
		},
		TradeSettlement: newTradeSettlement(l, ccy),
	}

	if l.Seller != nil {
		lineItem.Agreement.ItemSellerParty = newParty(l.Seller, ctx)
	}

	// BT-132: Purchase order line reference
	if l.Order != "" {
		lineItem.Agreement.OrderReference = &LineOrderReference{
			LineID: l.Order.String(),
		}
	}

	return lineItem
}

// newProduct builds the line's item (BG-31).
func newProduct(it *org.Item) *Product {
	p := &Product{
		Name: it.Name,
	}

	if it.Description != "" {
		p.Description = &it.Description
	}

	// BG-32: item attributes
	p.Characteristics = newCharacteristics(it.Attributes)

	for _, id := range it.Identities {
		// BT-157: Standard identifier (has scheme ID extension)
		if id.Ext.Has(iso.ExtKeySchemeID) {
			p.GlobalID = &GlobalID{
				SchemeID: id.Ext.Get(iso.ExtKeySchemeID).String(),
				Value:    id.Code.String(),
			}
		} else if id.Label != "" {
			// BT-158: Item classification (Label holds the list ID)
			p.Classification = &Classification{
				Code: &ListID{
					Value:  id.Code.String(),
					ListID: id.Label,
				},
			}
		} else if p.BuyerAssignedID == nil {
			// BT-156: Buyer's item identifier (plain identity)
			code := id.Code.String()
			p.BuyerAssignedID = &code
		}
	}

	return p
}

func newLineNotes(notes []*org.Note) []*Note {
	if len(notes) == 0 {
		return nil
	}
	out := make([]*Note, 0, len(notes))
	for _, n := range notes {
		note := &Note{Content: n.Text}
		if code := n.Ext.Get(untdid.ExtKeyTextSubject); code != "" {
			note.SubjectCode = code.String()
		}
		out = append(out, note)
	}
	return out
}

// newLineDocReference builds BT-128, the invoice line object identifier.
func newLineDocReference(id *org.Identity) *LineDocReference {
	if id == nil {
		return nil
	}
	ref := &LineDocReference{
		ID:       id.Code.String(),
		TypeCode: "130",
	}
	if id.Ext.Has(untdid.ExtKeyReference) {
		rc := id.Ext.Get(untdid.ExtKeyReference).String()
		ref.RefCode = &rc
	}
	return ref
}

func newTradeSettlement(l *bill.Line, ccy string) *TradeSettlement {
	var taxes []*Tax
	for _, tax := range l.Taxes {
		t := makeTaxCategory(tax)
		taxes = append(taxes, t)
	}

	stlm := &TradeSettlement{
		ApplicableTradeTax: taxes,
		Sum: &Summation{
			// BT-131: the line net amount, capped at the currency's
			// precision by BR-DEC-23.
			Amount: rescaleToCurrency(*l.Total, ccy),
		},
		Period:          newLinePeriod(l.Period),
		AllowanceCharge: newLineAllowanceCharges(l.Charges, l.Discounts, ccy),
	}

	// BT-133: Line buyer accounting reference
	if l.Cost != "" {
		stlm.AccountingAccount = &AccountingAccount{
			ID: l.Cost.String(),
		}
	}

	return stlm
}

// newLinePeriod writes BT-134/BT-135. Start and end are both optional, so only
// the ends the period actually carries are written out.
func newLinePeriod(p *cal.Period) *Period {
	if p == nil {
		return nil
	}
	out := &Period{}
	if d := documentDate(p.Start); d != nil {
		out.Start = &IssueDate{DateFormat: d}
	}
	if d := documentDate(p.End); d != nil {
		out.End = &IssueDate{DateFormat: d}
	}
	return out
}

// lineCurrency resolves the currency a line's amounts are expressed in: its
// item may override the document's.
func lineCurrency(inv *bill.Invoice, l *bill.Line) string {
	if l.Item != nil && l.Item.Currency != currency.CodeEmpty {
		return l.Item.Currency.String()
	}
	return inv.Currency.String()
}
