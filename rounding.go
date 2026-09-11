package cii

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// roundToCurrency recalculates an invoice using the currency rounding rule so
// that every monetary amount fits the number of decimals CII allows.
//
// EN 16931's BR-DEC-* rules cap almost all amounts — line net amounts (BT-131),
// tax bases (BT-116), document totals — at the currency's precision, two
// decimals for most currencies. GOBL's `precise` rounding rule keeps line
// totals at a higher precision, and since v0.505 RemoveIncludedTaxes switches a
// document away from `currency` rounding so that the tax-exclusive amounts it
// derives still add up to the original tax-inclusive ones. Emitting those line
// totals as-is breaks BR-DEC-23, and rounding them individually on the way out
// breaks BR-CO-10, as the rounded lines no longer sum to the document total.
//
// Recalculating instead rounds every amount consistently, and any difference
// against the amount originally payable is carried in the invoice's rounding
// total, which CII expresses as ram:RoundingAmount (BT-114).
func roundToCurrency(inv *bill.Invoice) error {
	if !exceedsCurrencyPrecision(inv) {
		return nil
	}
	payable := inv.Totals.Payable

	if inv.Tax == nil {
		inv.Tax = new(bill.Tax)
	}
	inv.Tax.Rounding = tax.RoundingRuleCurrency
	if err := inv.Calculate(); err != nil {
		return err
	}

	// Preserve the amount actually owed: rounding each line to the currency
	// may shift the total by a cent or two.
	diff := payable.Subtract(inv.Totals.Payable)
	if diff.IsZero() {
		return nil
	}
	rnd := diff
	if inv.Totals.Rounding != nil {
		rnd = inv.Totals.Rounding.Add(diff)
	}
	inv.Totals.Rounding = &rnd
	return inv.Calculate()
}

// exceedsCurrencyPrecision reports whether the invoice holds an amount with
// more decimals than its currency allows, and so cannot be written out as it
// stands. Only the amounts the calculator keeps at the rounding rule's
// precision are checked; the document totals are always rounded to the
// currency, and unit prices (BT-146, BT-148) are exempt from the BR-DEC rules.
//
// Invoices already within the currency's precision are left untouched, so a
// document that needs no rounding is never recalculated.
func exceedsCurrencyPrecision(inv *bill.Invoice) bool {
	if inv.Totals == nil {
		return false
	}
	def := inv.Currency.Def()
	if def == nil {
		return false
	}
	exp := def.Subunits

	over := func(a *num.Amount) bool {
		return a != nil && a.Exp() > exp
	}
	for _, l := range inv.Lines {
		if over(l.Total) || over(l.Sum) {
			return true
		}
		for _, d := range l.Discounts {
			if over(&d.Amount) {
				return true
			}
		}
		for _, c := range l.Charges {
			if over(&c.Amount) {
				return true
			}
		}
	}
	for _, d := range inv.Discounts {
		if over(&d.Amount) || over(d.Base) {
			return true
		}
	}
	for _, c := range inv.Charges {
		if over(&c.Amount) || over(c.Base) {
			return true
		}
	}
	if t := inv.Totals.Taxes; t != nil {
		for _, cat := range t.Categories {
			for _, r := range cat.Rates {
				if over(&r.Base) || over(&r.Amount) {
					return true
				}
			}
		}
	}
	return false
}

// rescaleToCurrency rounds the amount to the natural precision of the given
// currency code (2 for EUR, 0 for JPY). Falls back to the amount's existing
// precision if the currency code is unknown.
func rescaleToCurrency(a num.Amount, ccy string) string {
	if def := currency.Code(ccy).Def(); def != nil {
		return def.Rescale(a).String()
	}
	return a.String()
}
