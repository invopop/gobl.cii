package ctog

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) prepareChargesAndDiscounts(stlm *ApplicableHeaderTradeSettlement) error {
	var charges []*bill.Charge
	var discounts []*bill.Discount

	for _, ac := range stlm.SpecifiedTradeAllowanceCharge {
		if ac.ChargeIndicator.Indicator {
			c, err := newCharge(&ac)
			if err != nil {
				return err
			}
			if charges == nil {
				charges = make([]*bill.Charge, 0)
			}
			charges = append(charges, c)
		} else {
			d, err := newDiscount(&ac)
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

func newCharge(ac *SpecifiedTradeAllowanceCharge) (*bill.Charge, error) {
	// This is a charge
	c := &bill.Charge{}
	if ac.Reason != nil {
		c.Reason = *ac.Reason
	}
	if ac.ActualAmount != "" {
		c.Amount, _ = num.AmountFromString(ac.ActualAmount)
	}
	if ac.ReasonCode != nil {
		c.Code = cbc.Code(*ac.ReasonCode)
	}
	if ac.BasisAmount != nil {
		b, err := num.AmountFromString(*ac.BasisAmount)
		if err != nil {
			return nil, err
		}
		c.Base = &b
	}
	if ac.CalculationPercent != nil {
		if !strings.HasSuffix(*ac.CalculationPercent, "%") {
			*ac.CalculationPercent += "%"
		}
		p, err := num.PercentageFromString(*ac.CalculationPercent)
		if err != nil {
			return nil, err
		}
		c.Percent = &p
	}
	if ac.CategoryTradeTax.TypeCode != "" {
		c.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.CategoryTradeTax.TypeCode),
				Rate:     FindTaxKey(ac.CategoryTradeTax.CategoryCode),
			},
		}
	}
	// Format percentages
	if ac.CategoryTradeTax.RateApplicablePercent != nil {
		if !strings.HasSuffix(*ac.CategoryTradeTax.RateApplicablePercent, "%") {
			*ac.CategoryTradeTax.RateApplicablePercent += "%"
		}
		p, err := num.PercentageFromString(*ac.CategoryTradeTax.RateApplicablePercent)
		if err != nil {
			return nil, err
		}
		c.Taxes[0].Percent = &p
	}
	return c, nil
}

func newDiscount(ac *SpecifiedTradeAllowanceCharge) (*bill.Discount, error) {
	d := &bill.Discount{}
	if ac.Reason != nil {
		d.Reason = *ac.Reason
	}
	if ac.ActualAmount != "" {
		d.Amount, _ = num.AmountFromString(ac.ActualAmount)
	}
	if ac.ReasonCode != nil {
		d.Code = cbc.Code(*ac.ReasonCode)
	}
	if ac.BasisAmount != nil {
		b, err := num.AmountFromString(*ac.BasisAmount)
		if err != nil {
			return nil, err
		}
		d.Base = &b
	}
	if ac.CalculationPercent != nil {
		if !strings.HasSuffix(*ac.CalculationPercent, "%") {
			*ac.CalculationPercent += "%"
		}
		p, err := num.PercentageFromString(*ac.CalculationPercent)
		if err != nil {
			return nil, err
		}
		d.Percent = &p
	}
	if ac.CategoryTradeTax.TypeCode != "" {
		d.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.CategoryTradeTax.TypeCode),
				Rate:     FindTaxKey(ac.CategoryTradeTax.CategoryCode),
			},
		}
	}
	if ac.CategoryTradeTax.RateApplicablePercent != nil {
		if !strings.HasSuffix(*ac.CategoryTradeTax.RateApplicablePercent, "%") {
			*ac.CategoryTradeTax.RateApplicablePercent += "%"
		}
		p, err := num.PercentageFromString(*ac.CategoryTradeTax.RateApplicablePercent)
		if err != nil {
			return nil, err
		}
		d.Taxes[0].Percent = &p
	}
	return d, nil
}
