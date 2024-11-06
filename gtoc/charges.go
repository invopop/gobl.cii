package gtoc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
)

func newAllowanceCharges(inv *bill.Invoice) []*AllowanceCharge {
	if inv.Charges == nil && inv.Discounts == nil {
		return nil
	}
	ac := make([]*AllowanceCharge, len(inv.Charges)+len(inv.Discounts))
	for i, charge := range inv.Charges {
		ac[i] = newCharge(charge)
	}
	for i, discount := range inv.Discounts {
		ac[i+len(inv.Charges)] = newDiscount(discount)
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

func newCharge(charge *bill.Charge) *AllowanceCharge {
	c := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: true},
		Amount:          charge.Amount.String(),
	}
	if charge.Reason != "" {
		c.Reason = charge.Reason
	}
	if charge.Code != "" {
		c.ReasonCode = charge.Code.String()
	}
	if charge.Percent != nil {
		p := charge.Percent.String()
		c.Percent = p
	}
	if charge.Taxes != nil {
		c.Tax = makeTaxCategory(charge.Taxes[0])
	}
	return c
}

func newDiscount(discount *bill.Discount) *AllowanceCharge {
	d := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: false},
		Amount:          discount.Amount.String(),
	}
	if discount.Reason != "" {
		d.Reason = discount.Reason
	}
	if discount.Code != "" {
		d.ReasonCode = discount.Code.String()
	}
	if discount.Percent != nil {
		p := discount.Percent.String()
		d.Percent = p
	}
	if discount.Taxes != nil {
		d.Tax = makeTaxCategory(discount.Taxes[0])
	}
	return d
}

func makeLineCharge(charge *bill.LineCharge) *AllowanceCharge {
	c := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: true},
		Amount:          charge.Amount.String(),
	}
	if charge.Reason != "" {
		c.Reason = charge.Reason
	}
	if charge.Code != "" {
		c.ReasonCode = charge.Code.String()
	}
	if charge.Percent != nil {
		p := charge.Percent.String()
		c.Percent = p
	}
	return c
}

func makeLineDiscount(discount *bill.LineDiscount) *AllowanceCharge {
	d := &AllowanceCharge{
		ChargeIndicator: Indicator{Value: false},
		Amount:          discount.Amount.String(),
	}
	if discount.Reason != "" {
		d.Reason = discount.Reason
	}
	if discount.Code != "" {
		d.ReasonCode = discount.Code.String()
	}
	if discount.Percent != nil {
		p := discount.Percent.String()
		d.Percent = p
	}
	return d
}

func makeTaxCategory(tax *tax.Combo) *Tax {
	category := &Tax{}
	if tax.Category != "" {
		category.TypeCode = tax.Category.String()
	}
	if tax.Percent != nil {
		category.RateApplicablePercent = tax.Percent.StringWithoutSymbol()
	}
	return category
}
