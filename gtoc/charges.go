package gtoc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
)

func newCharges(inv *bill.Invoice) []*AllowanceCharge {
	if inv.Charges == nil && inv.Discounts == nil {
		return nil
	}
	ac := make([]*AllowanceCharge, len(inv.Charges)+len(inv.Discounts))
	for i, charge := range inv.Charges {
		ac[i] = makeCharge(charge)
	}
	for i, discount := range inv.Discounts {
		ac[i+len(inv.Charges)] = makeDiscount(discount)
	}
	return ac
}

func makeCharge(charge *bill.Charge) *AllowanceCharge {
	c := &AllowanceCharge{
		ChargeIndicator: true,
		Amount:          charge.Amount.String(),
	}
	if charge.Reason != "" {
		c.Reason = charge.Reason
	}
	if charge.Code != "" {
		c.ReasonCode = charge.Code
	}
	if charge.Percent != nil {
		p := charge.Percent.String()
		c.Percent = p
	}
	if charge.Taxes != nil {
		c.Tax = makeTaxCategory(charge.Taxes)
	}
	return c
}

func makeDiscount(discount *bill.Discount) *AllowanceCharge {
	d := &AllowanceCharge{
		ChargeIndicator: false,
		Amount:          discount.Amount.String(),
	}
	if discount.Reason != "" {
		d.Reason = discount.Reason
	}
	if discount.Code != "" {
		d.ReasonCode = discount.Code
	}
	if discount.Percent != nil {
		p := discount.Percent.String()
		d.Percent = p
	}
	if discount.Taxes != nil {
		d.Tax = makeTaxCategory(discount.Taxes)
	}
	return d
}

func makeTaxCategory(taxes tax.Set) *ApplicableTradeTax {
	category := &ApplicableTradeTax{}
	if taxes[0].Category != "" {
		category.TaxCode = taxes[0].Category.String()
	}
	if taxes[0].Percent != nil {
		category.TaxRatePercent = taxes[0].Percent.StringWithoutSymbol()
	}
	return category
}
