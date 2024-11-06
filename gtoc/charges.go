package gtoc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/tax"
)

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
		Amount:          c.Amount.String(),
	}
	if c.Reason != "" {
		ac.Reason = c.Reason
	}
	if c.Ext[untdid.ExtKeyCharge] != "" {
		ac.ReasonCode = c.Ext[untdid.ExtKeyCharge].String()
	}
	if c.Percent != nil {
		p := c.Percent.String()
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
		Amount:          d.Amount.String(),
	}
	if d.Reason != "" {
		ac.Reason = d.Reason
	}
	if d.Ext[untdid.ExtKeyAllowance] != "" {
		ac.ReasonCode = d.Ext[untdid.ExtKeyAllowance].String()
	}
	if d.Percent != nil {
		p := d.Percent.String()
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
		Amount:          c.Amount.String(),
	}
	if c.Reason != "" {
		ac.Reason = c.Reason
	}
	if c.Ext[untdid.ExtKeyCharge] != "" {
		ac.ReasonCode = c.Ext[untdid.ExtKeyCharge].String()
	}
	if c.Percent != nil {
		p := c.Percent.String()
		ac.Percent = p
	}
	return ac
}

func makeLineDiscount(d *bill.LineDiscount) *AllowanceCharge {
	ac := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: false},
		Amount:          d.Amount.String(),
	}
	if d.Reason != "" {
		ac.Reason = d.Reason
	}
	if d.Ext[untdid.ExtKeyAllowance] != "" {
		ac.ReasonCode = d.Ext[untdid.ExtKeyAllowance].String()
	}
	if d.Percent != nil {
		p := d.Percent.String()
		ac.Percent = p
	}
	return ac
}

func makeTaxCategory(tax *tax.Combo) *Tax {
	c := &Tax{}
	if tax.Category != "" {
		c.TypeCode = tax.Category.String()
	}
	if tax.Ext[untdid.ExtKeyTaxCategory] != "" {
		c.CategoryCode = tax.Ext[untdid.ExtKeyTaxCategory].String()
	}
	if tax.Percent != nil {
		c.RateApplicablePercent = tax.Percent.StringWithoutSymbol()
	}
	return c
}
