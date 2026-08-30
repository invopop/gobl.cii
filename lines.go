package cii

import (
	"fmt"
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/tax"
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
	// ParentLineID (BT-X-304 / EXT-FR-FE-162) references the LineID of the
	// parent line in an EXTENDED-CTC-FR line hierarchy.
	ParentLineID string `xml:"ram:ParentLineID,omitempty"`
	// LineStatusReasonCode (BT-X-8 / EXT-FR-FE-163) carries the line subtype
	// in a hierarchy: GROUP (aggregating header) or DETAIL (leaf). XSD order
	// places it after the optional LineStatusCode, which we never emit.
	LineStatusReasonCode string  `xml:"ram:LineStatusReasonCode,omitempty"`
	Note                 []*Note `xml:"ram:IncludedNote,omitempty"`
}

// LineAgreement defines the structure of the SpecifiedLineTradeAgreement in the CII standard
type LineAgreement struct {
	OrderReference      *LineOrderReference `xml:"ram:BuyerOrderReferencedDocument,omitempty"`
	AdditionalReference *LineDocReference   `xml:"ram:AdditionalReferencedDocument,omitempty"`
	NetPrice            *NetPrice           `xml:"ram:NetPriceProductTradePrice"`
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

// Line subtype codes (BT-X-8 / EXT-FR-FE-163) for the EXTENDED-CTC-FR
// line hierarchy.
const (
	lineSubtypeGroup  = "GROUP"
	lineSubtypeDetail = "DETAIL"
)

func (out *Invoice) addLines(lines []*bill.Line, ctx Context) error {
	// The EXTENDED-CTC-FR profile is the only CII context that supports a
	// line hierarchy (parent/child via ParentLineID + subtype). For every
	// other guideline a Breakdown is informational only and is not emitted,
	// preserving the prior flat behaviour.
	hierarchy := ctx.Is(ContextPeppolFranceFacturXV1)

	var Lines []*Line
	for _, l := range lines {
		if hierarchy && len(l.Breakdown) > 0 {
			Lines = append(Lines, newHierarchyLines(l)...)
			continue
		}
		if line := newLine(l); line != nil {
			Lines = append(Lines, line)
		}
	}

	out.Transaction.Lines = Lines
	return nil
}

// newHierarchyLines maps a bill.Line carrying a Breakdown onto an
// EXTENDED-CTC-FR line hierarchy: the line itself becomes a GROUP header
// (aggregating, carries no VAT so it is excluded from the VAT breakdown
// and the BT-106 sum) whose BT-131 equals the sum of its children, and
// each SubLine becomes a DETAIL child carrying the parent line's VAT
// category (SubLines have no taxes of their own) and pointing back via
// ParentLineID. See BR-FREXT-06 (subtype required when a parent is set)
// and BR-FREXT-08 (GROUP net = Σ children).
func newHierarchyLines(l *bill.Line) []*Line {
	group := newLine(l)
	if group == nil {
		return nil
	}
	group.LineDoc.LineStatusReasonCode = lineSubtypeGroup
	if group.TradeSettlement != nil {
		group.TradeSettlement.ApplicableTradeTax = nil
	}

	lines := []*Line{group}
	parentID := group.LineDoc.ID
	for i, sub := range l.Breakdown {
		if sub == nil || sub.Item == nil {
			continue
		}
		child := newLine(subLineAsLine(sub, l.Taxes, i+1))
		if child == nil {
			continue
		}
		child.LineDoc.ID = fmt.Sprintf("%s.%d", parentID, i+1)
		child.LineDoc.ParentLineID = parentID
		child.LineDoc.LineStatusReasonCode = lineSubtypeDetail
		lines = append(lines, child)
	}
	return lines
}

// subLineAsLine adapts a bill.SubLine into a bill.Line so it can reuse
// newLine. SubLines carry no taxes of their own, so the parent line's tax
// set is inherited — the whole hierarchy shares one VAT category.
func subLineAsLine(sub *bill.SubLine, taxes tax.Set, index int) *bill.Line {
	return &bill.Line{
		Index:      index,
		Quantity:   sub.Quantity,
		Item:       sub.Item,
		Identifier: sub.Identifier,
		Period:     sub.Period,
		Order:      sub.Order,
		Cost:       sub.Cost,
		Discounts:  sub.Discounts,
		Charges:    sub.Charges,
		Notes:      sub.Notes,
		Taxes:      taxes,
		Total:      sub.Total,
	}
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

	// BT-155: Seller's item identifier
	if it.Ref != "" {
		ref := it.Ref.String()
		lineItem.Product.SellerAssignedID = &ref
	}

	// BT-159: Item country of origin
	if it.Origin != "" {
		origin := string(it.Origin)
		lineItem.Product.Origin = &origin
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
			// BT-157: Standard identifier (has scheme ID extension)
			if id.Ext.Has(iso.ExtKeySchemeID) {
				lineItem.Product.GlobalID = &GlobalID{
					SchemeID: id.Ext.Get(iso.ExtKeySchemeID).String(),
					Value:    id.Code.String(),
				}
			} else if id.Label != "" {
				// BT-158: Item classification (Label holds the list ID)
				lineItem.Product.Classification = &Classification{
					Code: &ListID{
						Value:  id.Code.String(),
						ListID: id.Label,
					},
				}
			} else if lineItem.Product.BuyerAssignedID == nil {
				// BT-156: Buyer's item identifier (plain identity)
				code := id.Code.String()
				lineItem.Product.BuyerAssignedID = &code
			}
		}
	}

	// BT-128: Invoice line object identifier
	if l.Identifier != nil {
		ref := &LineDocReference{
			ID:       l.Identifier.Code.String(),
			TypeCode: "130",
		}
		if l.Identifier.Ext.Has(untdid.ExtKeyReference) {
			rc := l.Identifier.Ext.Get(untdid.ExtKeyReference).String()
			ref.RefCode = &rc
		}
		lineItem.Agreement.AdditionalReference = ref
	}

	// BT-132: Purchase order line reference
	if l.Order != "" {
		lineItem.Agreement.OrderReference = &LineOrderReference{
			LineID: l.Order.String(),
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

	if l.Period != nil {
		stlm.Period = &Period{
			Start: &IssueDate{
				DateFormat: &Date{
					Value:  formatIssueDate(l.Period.Start),
					Format: issueDateFormat,
				},
			},
			End: &IssueDate{
				DateFormat: &Date{
					Value:  formatIssueDate(l.Period.End),
					Format: issueDateFormat,
				},
			},
		}
	}

	if len(l.Charges) > 0 || len(l.Discounts) > 0 {
		stlm.AllowanceCharge = newLineAllowanceCharges(l)
	}

	// BT-133: Line buyer accounting reference
	if l.Cost != "" {
		stlm.AccountingAccount = &AccountingAccount{
			ID: l.Cost.String(),
		}
	}

	return stlm
}
