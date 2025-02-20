package gtoc

import (
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

func newAllowanceCharges(inv *bill.Invoice) []*document.AllowanceCharge {
	if inv.Charges == nil && inv.Discounts == nil {
		return nil
	}
	ac := make([]*document.AllowanceCharge, len(inv.Charges)+len(inv.Discounts))
	sum := inv.Totals.Sum
	for i, c := range inv.Charges {
		ac[i] = newCharge(c, sum)
	}
	for i, d := range inv.Discounts {
		ac[i+len(inv.Charges)] = newDiscount(d, sum)
	}
	return ac
}

func newLineAllowanceCharges(line *bill.Line) []*document.AllowanceCharge {
	if line.Charges == nil && line.Discounts == nil {
		return nil
	}
	ac := make([]*document.AllowanceCharge, len(line.Charges)+len(line.Discounts))
	for i, charge := range line.Charges {
		ac[i] = makeLineCharge(charge)
	}
	for i, discount := range line.Discounts {
		ac[i+len(line.Charges)] = makeLineDiscount(discount)
	}
	return ac
}

func newCharge(c *bill.Charge, base num.Amount) *document.AllowanceCharge {
	ac := &document.AllowanceCharge{
		ChargeIndicator: document.Indicator{Value: true},
		Amount:          c.Amount.Rescale(2).String(),
	}

	if c.Base != nil {
		ac.Base = c.Base.Rescale(2).String()
	} else {
		ac.Base = base.Rescale(2).String()
	}

	if c.Reason != "" {
		ac.Reason = c.Reason
	}
	ac.ReasonCode = c.Ext.Get(untdid.ExtKeyCharge).String()
	if c.Percent != nil {
		p := c.Percent.StringWithoutSymbol()
		ac.Percent = p
	}
	if c.Taxes != nil {
		ac.Tax = makeTaxCategory(c.Taxes[0])
	}
	return ac
}

func newDiscount(d *bill.Discount, base num.Amount) *document.AllowanceCharge {
	ac := &document.AllowanceCharge{
		ChargeIndicator: document.Indicator{Value: false},
		Amount:          d.Amount.Rescale(2).String(),
	}

	if d.Base != nil {
		ac.Base = d.Base.Rescale(2).String()
	} else {
		ac.Base = base.Rescale(2).String()
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

func makeLineCharge(c *bill.LineCharge) *document.AllowanceCharge {
	ac := &document.AllowanceCharge{
		ChargeIndicator: document.Indicator{Value: true},
		Amount:          c.Amount.Rescale(2).String(),
	}
	if c.Reason != "" {
		ac.Reason = c.Reason
	}
	ac.ReasonCode = c.Ext.Get(untdid.ExtKeyCharge).String()
	if c.Percent != nil {
		p := c.Percent.String()
		ac.Percent = p
	}
	return ac
}

func makeLineDiscount(d *bill.LineDiscount) *document.AllowanceCharge {
	ac := &document.AllowanceCharge{
		ChargeIndicator: document.Indicator{Value: false},
		Amount:          d.Amount.Rescale(2).String(),
	}
	if d.Reason != "" {
		ac.Reason = d.Reason
	}
	ac.ReasonCode = d.Ext.Get(untdid.ExtKeyAllowance).String()
	if d.Percent != nil {
		p := d.Percent.String()
		ac.Percent = p
	}
	return ac
}

func makeTaxCategory(tax *tax.Combo) *document.Tax {
	c := &document.Tax{}
	if tax.Category != "" {
		c.TypeCode = tax.Category.String()
	}
	c.CategoryCode = tax.Ext.Get(untdid.ExtKeyTaxCategory).String()
	if tax.Percent != nil {
		c.RateApplicablePercent = tax.Percent.StringWithoutSymbol()
	}
	return c
}
