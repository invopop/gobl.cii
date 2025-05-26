package cii

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/tax"
)

// AllowanceCharge defines the structure of SpecifiedTradeAllowanceCharge of the CII standard, also used for line items
type AllowanceCharge struct {
	ChargeIndicator Indicator `xml:"ram:ChargeIndicator"`
	Percent         string    `xml:"ram:CalculationPercent,omitempty"`
	Base            string    `xml:"ram:BasisAmount,omitempty"`
	Amount          string    `xml:"ram:ActualAmount,omitempty"`
	ReasonCode      string    `xml:"ram:ReasonCode,omitempty"`
	Reason          string    `xml:"ram:Reason,omitempty"`
	Tax             *Tax      `xml:"ram:CategoryTradeTax,omitempty"`
}

// Indicator defines the structure of Indicator of the CII standard
type Indicator struct {
	Value bool `xml:"udt:Indicator"`
}

func newAllowanceCharges(inv *bill.Invoice) []*AllowanceCharge {
	if inv.Charges == nil && inv.Discounts == nil {
		return nil
	}
	ac := make([]*AllowanceCharge, len(inv.Charges)+len(inv.Discounts))
	for i, c := range inv.Charges {
		ac[i] = newCharge(c)
	}
	for i, d := range inv.Discounts {
		ac[i+len(inv.Charges)] = newDiscount(d)
	}
	return ac
}

func newLineAllowanceCharges(line *bill.Line) []*AllowanceCharge {
	if line.Charges == nil && line.Discounts == nil {
		return nil
	}
	ac := make([]*AllowanceCharge, len(line.Charges)+len(line.Discounts))
	for i, charge := range line.Charges {
		ac[i] = makeLineCharge(charge)
	}
	for i, discount := range line.Discounts {
		ac[i+len(line.Charges)] = makeLineDiscount(discount)
	}
	return ac
}

func newCharge(c *bill.Charge) *AllowanceCharge {
	ac := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: true},
		Amount:          c.Amount.Rescale(2).String(),
	}

	if c.Base != nil {
		ac.Base = c.Base.Rescale(2).String()
	}

	if c.Reason != "" {
		ac.Reason = c.Reason
	}

	// ReasonCode is not supported at the moment due to a bug in the CII Schema that does not
	// have the correct UNTDID 7161 code list and subsequently rejects the invoice
	// if the reason code is mapped.
	// ac.ReasonCode = c.Ext.Get(untdid.ExtKeyCharge).String()

	if c.Percent != nil {
		p := c.Percent.StringWithoutSymbol()
		ac.Percent = p
	}
	if c.Taxes != nil {
		ac.Tax = makeTaxCategory(c.Taxes[0])
	}
	return ac
}

func newDiscount(d *bill.Discount) *AllowanceCharge {
	ac := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: false},
		Amount:          d.Amount.Rescale(2).String(),
	}

	if d.Base != nil {
		ac.Base = d.Base.Rescale(2).String()
	}

	if d.Reason != "" {
		ac.Reason = d.Reason
	}
	ac.ReasonCode = d.Ext.Get(untdid.ExtKeyAllowance).String()
	if d.Percent != nil {
		p := d.Percent.StringWithoutSymbol()
		ac.Percent = p
	}
	if d.Taxes != nil {
		ac.Tax = makeTaxCategory(d.Taxes[0])
	}

	return ac
}

func makeLineCharge(c *bill.LineCharge) *AllowanceCharge {
	ac := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: true},
		Amount:          c.Amount.Rescale(2).String(),
	}
	if c.Reason != "" {
		ac.Reason = c.Reason
	}
	ac.ReasonCode = c.Ext.Get(untdid.ExtKeyCharge).String()
	if c.Percent != nil && c.Base != nil {
		p := c.Percent.StringWithoutSymbol()
		ac.Percent = p
		ac.Base = c.Base.Rescale(2).String()
	}
	return ac
}

func makeLineDiscount(d *bill.LineDiscount) *AllowanceCharge {
	ac := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: false},
		Amount:          d.Amount.Rescale(2).String(),
	}
	if d.Reason != "" {
		ac.Reason = d.Reason
	}
	ac.ReasonCode = d.Ext.Get(untdid.ExtKeyAllowance).String()
	if d.Percent != nil && d.Base != nil {
		p := d.Percent.StringWithoutSymbol()
		ac.Percent = p
		ac.Base = d.Base.Rescale(2).String()
	}
	return ac
}

func makeTaxCategory(t *tax.Combo) *Tax {
	c := new(Tax)
	if t.Category != "" {
		c.TypeCode = t.Category.String()
	}
	cat := t.Ext.Get(untdid.ExtKeyTaxCategory)
	c.CategoryCode = cat.String()
	c.RateApplicablePercent = "0"
	if t.Percent != nil {
		c.RateApplicablePercent = t.Percent.StringWithoutSymbol()
	}
	if cat == en16931.TaxCategoryExempt {
		c.ExemptionReasonCode = t.Ext.Get(cef.ExtKeyVATEX).String()
	}
	return c
}
