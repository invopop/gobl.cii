package cii

import (
	"strings"

	"github.com/invopop/gobl.cii/structs"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// ParseAllowanceCharges extracts the charges logic from the CII document
func ParseCtoGCharges(settlement *structs.ApplicableHeaderTradeSettlement) ([]*bill.Charge, []*bill.Discount) {
	var charges []*bill.Charge
	var discounts []*bill.Discount

	for _, allowanceCharge := range settlement.SpecifiedTradeAllowanceCharge {
		if allowanceCharge.ChargeIndicator.Indicator {
			// This is a charge
			charge := &bill.Charge{}
			if allowanceCharge.Reason != nil {
				charge.Reason = *allowanceCharge.Reason
			}
			if allowanceCharge.ActualAmount != "" {
				charge.Amount, _ = num.AmountFromString(allowanceCharge.ActualAmount)
			}
			if allowanceCharge.ReasonCode != nil {
				charge.Code = *allowanceCharge.ReasonCode
			}
			if allowanceCharge.BasisAmount != nil {
				basis, _ := num.AmountFromString(*allowanceCharge.BasisAmount)
				charge.Base = &basis
			}
			if allowanceCharge.CalculationPercent != nil {
				if !strings.HasSuffix(*allowanceCharge.CalculationPercent, "%") {
					*allowanceCharge.CalculationPercent += "%"
				}
				percent, _ := num.PercentageFromString(*allowanceCharge.CalculationPercent)
				charge.Percent = &percent
			}
			if allowanceCharge.CategoryTradeTax.TypeCode != "" {
				charge.Taxes = tax.Set{
					{
						Category: cbc.Code(allowanceCharge.CategoryTradeTax.TypeCode),
						Rate:     FindTaxKey(allowanceCharge.CategoryTradeTax.CategoryCode),
					},
				}
			}
			// Format percentages
			if allowanceCharge.CategoryTradeTax.RateApplicablePercent != nil {
				if !strings.HasSuffix(*allowanceCharge.CategoryTradeTax.RateApplicablePercent, "%") {
					*allowanceCharge.CategoryTradeTax.RateApplicablePercent += "%"
				}
				percent, _ := num.PercentageFromString(*allowanceCharge.CategoryTradeTax.RateApplicablePercent)
				charge.Taxes[0].Percent = &percent
			}
			if charges == nil {
				charges = make([]*bill.Charge, 0)
			}
			charges = append(charges, charge)
		} else {
			// This is a discount
			discount := &bill.Discount{}
			if allowanceCharge.Reason != nil {
				discount.Reason = *allowanceCharge.Reason
			}
			if allowanceCharge.ActualAmount != "" {
				discount.Amount, _ = num.AmountFromString(allowanceCharge.ActualAmount)
			}
			if allowanceCharge.ReasonCode != nil {
				discount.Code = *allowanceCharge.ReasonCode
			}
			if allowanceCharge.BasisAmount != nil {
				basis, _ := num.AmountFromString(*allowanceCharge.BasisAmount)
				discount.Base = &basis
			}
			if allowanceCharge.CalculationPercent != nil {
				if !strings.HasSuffix(*allowanceCharge.CalculationPercent, "%") {
					*allowanceCharge.CalculationPercent += "%"
				}
				percent, _ := num.PercentageFromString(*allowanceCharge.CalculationPercent)
				discount.Percent = &percent
			}
			if allowanceCharge.CategoryTradeTax.TypeCode != "" {
				discount.Taxes = tax.Set{
					{
						Category: cbc.Code(allowanceCharge.CategoryTradeTax.TypeCode),
						Rate:     FindTaxKey(allowanceCharge.CategoryTradeTax.CategoryCode),
					},
				}
			}
			if allowanceCharge.CategoryTradeTax.RateApplicablePercent != nil {
				if !strings.HasSuffix(*allowanceCharge.CategoryTradeTax.RateApplicablePercent, "%") {
					*allowanceCharge.CategoryTradeTax.RateApplicablePercent += "%"
				}
				percent, _ := num.PercentageFromString(*allowanceCharge.CategoryTradeTax.RateApplicablePercent)
				discount.Taxes[0].Percent = &percent
			}
			if discounts == nil {
				discounts = make([]*bill.Discount, 0)
			}
			discounts = append(discounts, discount)
		}
	}

	return charges, discounts
}
