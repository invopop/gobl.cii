package ctog

import (
	"strings"

	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) prepareChargesAndDiscounts(stlm *document.Settlement) error {
	var charges []*bill.Charge
	var discounts []*bill.Discount

	for _, ac := range stlm.AllowanceCharges {
		if ac.ChargeIndicator.Value {
			c, err := newCharge(ac)
			if err != nil {
				return err
			}
			if charges == nil {
				charges = make([]*bill.Charge, 0)
			}
			charges = append(charges, c)
		} else {
			d, err := newDiscount(ac)
			if err != nil {
				return err
			}
			if discounts == nil {
				discounts = make([]*bill.Discount, 0)
			}
			discounts = append(discounts, d)
		}
	}
	if charges != nil {
		c.inv.Charges = charges
	}
	if discounts != nil {
		c.inv.Discounts = discounts
	}
	return nil
}

func newCharge(ac *document.AllowanceCharge) (*bill.Charge, error) {
	// This is a charge
	c := &bill.Charge{}
	if ac.Reason != "" {
		c.Reason = ac.Reason
	}
	if ac.Amount != "" {
		c.Amount, _ = num.AmountFromString(ac.Amount)
	}
	if ac.ReasonCode != "" {
		c.Ext = tax.Extensions{
			untdid.ExtKeyCharge: cbc.Code(ac.ReasonCode),
		}
	}
	if ac.Base != "" {
		b, err := num.AmountFromString(ac.Base)
		if err != nil {
			return nil, err
		}
		c.Base = &b
	}
	if ac.Percent != "" {
		if !strings.HasSuffix(ac.Percent, "%") {
			ac.Percent += "%"
		}
		p, err := num.PercentageFromString(ac.Percent)
		if err != nil {
			return nil, err
		}
		c.Percent = &p
	}
	if ac.Tax.TypeCode != "" {
		c.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.Tax.TypeCode),
			},
		}
	}
	// Format percentages
	if ac.Tax.RateApplicablePercent != "" {
		if !strings.HasSuffix(ac.Tax.RateApplicablePercent, "%") {
			ac.Tax.RateApplicablePercent += "%"
		}
		p, err := num.PercentageFromString(ac.Tax.RateApplicablePercent)
		if err != nil {
			return nil, err
		}
		c.Taxes[0].Percent = &p
	}
	return c, nil
}

func newDiscount(ac *document.AllowanceCharge) (*bill.Discount, error) {
	d := &bill.Discount{}
	if ac.Reason != "" {
		d.Reason = ac.Reason
	}
	if ac.Amount != "" {
		d.Amount, _ = num.AmountFromString(ac.Amount)
	}
	if ac.ReasonCode != "" {
		d.Ext = tax.Extensions{
			untdid.ExtKeyAllowance: cbc.Code(ac.ReasonCode),
		}
	}
	if ac.Base != "" {
		b, err := num.AmountFromString(ac.Base)
		if err != nil {
			return nil, err
		}
		d.Base = &b
	}
	if ac.Percent != "" {
		if !strings.HasSuffix(ac.Percent, "%") {
			ac.Percent += "%"
		}
		p, err := num.PercentageFromString(ac.Percent)
		if err != nil {
			return nil, err
		}
		d.Percent = &p
	}
	if ac.Tax.TypeCode != "" {
		d.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.Tax.TypeCode),
			},
		}
	}
	if ac.Tax.RateApplicablePercent != "" {
		if !strings.HasSuffix(ac.Tax.RateApplicablePercent, "%") {
			ac.Tax.RateApplicablePercent += "%"
		}
		p, err := num.PercentageFromString(ac.Tax.RateApplicablePercent)
		if err != nil {
			return nil, err
		}
		d.Taxes[0].Percent = &p
	}
	return d, nil
}

func getLineCharge(ac *document.AllowanceCharge) (*bill.LineCharge, error) {
	a, err := num.AmountFromString(ac.Amount)
	if err != nil {
		return nil, err
	}
	c := &bill.LineCharge{
		Amount: a,
	}
	if ac.ReasonCode != "" {
		c.Ext = tax.Extensions{
			untdid.ExtKeyCharge: cbc.Code(ac.ReasonCode),
		}
	}
	if ac.Reason != "" {
		c.Reason = ac.Reason
	}
	if ac.Percent != "" {
		if !strings.HasSuffix(ac.Percent, "%") {
			ac.Percent += "%"
		}
		p, err := num.PercentageFromString(ac.Percent)
		if err != nil {
			return nil, err
		}
		c.Percent = &p
	}
	return c, nil
}

func getLineDiscount(ac *document.AllowanceCharge) (*bill.LineDiscount, error) {
	a, err := num.AmountFromString(ac.Amount)
	if err != nil {
		return nil, err
	}
	d := &bill.LineDiscount{
		Amount: a,
	}
	if ac.ReasonCode != "" {
		d.Ext = tax.Extensions{
			untdid.ExtKeyAllowance: cbc.Code(ac.ReasonCode),
		}
	}
	if ac.Reason != "" {
		d.Reason = ac.Reason
	}
	if ac.Percent != "" {
		if !strings.HasSuffix(ac.Percent, "%") {
			ac.Percent += "%"
		}
		p, err := num.PercentageFromString(ac.Percent)
		if err != nil {
			return nil, err
		}
		d.Percent = &p
	}
	return d, nil
}
