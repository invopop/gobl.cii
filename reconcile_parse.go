package cii

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// goblReconcileTotals aligns the document with the amounts the sender declared.
// BT-131 is mandatory (BR-24) and the totals are summed from it (BR-CO-10,
// BR-CO-13), while BT-149 is optional and in no rule, so BT-131 decides between
// them. What reconciles under no reading is tagged for bypass and recorded as
// sent.
func goblReconcileTotals(in *Invoice, out *bill.Invoice) error {
	goblApplyRounding(in, out)

	if err := out.Calculate(); err != nil {
		return err
	}

	if swaps := goblDropConflictingBaseQuantities(in, out); len(swaps) > 0 {
		if err := out.Calculate(); err != nil {
			return err
		}
		// A line that matches neither reading keeps the one the standard
		// describes; dropping its base quantity bought us nothing.
		if goblRestoreBaseQuantities(in, out, swaps) {
			if err := out.Calculate(); err != nil {
				return err
			}
		}
	}

	if goblDeclaredTotalsAgree(in, out) {
		return nil
	}
	return goblApplyDeclaredTotals(in, out)
}

// priceSwap records a line whose base quantity was dropped, so the change can
// be undone if it did not reconcile the line.
type priceSwap struct {
	index int
	price num.Amount
}

// goblDropConflictingBaseQuantities removes the base quantity from any line that
// only matches its declared amount (BT-131) without it. Lines matching either
// way keep it, dividing being the reading the standard describes.
func goblDropConflictingBaseQuantities(in *Invoice, out *bill.Invoice) []priceSwap {
	var swaps []priceSwap
	for i, docLine := range in.Transaction.Lines {
		if i >= len(out.Lines) {
			break
		}
		line := out.Lines[i]
		if line == nil || line.Item == nil || line.Item.Price == nil || line.Total == nil {
			continue
		}
		np := docLine.Agreement.NetPrice
		if np == nil || np.BaseQuantity == nil || np.BaseQuantity.Amount == "" {
			continue
		}
		declared, ok := goblDeclaredLineAmount(docLine)
		if !ok || declared.Equals(*line.Total) {
			continue
		}
		// Retry without the base quantity, the only other reading available.
		price, err := num.AmountFromString(np.Amount)
		if err != nil {
			continue
		}
		swaps = append(swaps, priceSwap{index: i, price: *line.Item.Price})
		line.Item.Price = &price
	}
	return swaps
}

// goblRestoreBaseQuantities restores the standard price reading on swapped lines
// that still do not match.
func goblRestoreBaseQuantities(in *Invoice, out *bill.Invoice, swaps []priceSwap) bool {
	lines := in.Transaction.Lines
	restored := false
	for _, s := range swaps {
		if s.index >= len(out.Lines) || s.index >= len(lines) {
			continue
		}
		line := out.Lines[s.index]
		if line == nil || line.Item == nil || line.Total == nil {
			continue
		}
		declared, ok := goblDeclaredLineAmount(lines[s.index])
		if ok && declared.Equals(*line.Total) {
			continue
		}
		price := s.price
		line.Item.Price = &price
		restored = true
	}
	return restored
}

// goblDeclaredTotalsAgree reports whether the calculated amounts reproduce every
// declared one: BT-131 per line, then BT-106, BT-109, BT-110, BT-112 and BT-115.
func goblDeclaredTotalsAgree(in *Invoice, out *bill.Invoice) bool {
	for i, docLine := range in.Transaction.Lines {
		if i >= len(out.Lines) {
			return false
		}
		line := out.Lines[i]
		if line == nil || line.Total == nil {
			continue
		}
		declared, ok := goblDeclaredLineAmount(docLine)
		if ok && !declared.Equals(*line.Total) {
			return false
		}
	}

	t := out.Totals
	s := goblSummary(in)
	if t == nil || s == nil {
		return false
	}
	pairs := []struct {
		declared string
		computed num.Amount
	}{
		{s.LineTotalAmount, t.Sum},
		{s.TaxBasisTotalAmount, t.Total},
		{s.GrandTotalAmount, t.TotalWithTax},
		{s.DuePayableAmount, goblPayableAmount(t)},
	}
	for _, p := range pairs {
		declared, ok := goblDeclaredAmount(p.declared)
		if ok && !declared.Equals(p.computed) {
			return false
		}
	}

	if s.TaxTotalAmount != nil {
		if declared, ok := goblDeclaredAmount(s.TaxTotalAmount.Amount); ok && !declared.Equals(t.Tax) {
			return false
		}
	}
	return true
}

// goblPayableAmount returns what BT-115 is built from: Due once advances are
// deducted, Payable otherwise, matching the outbound mapping.
func goblPayableAmount(t *bill.Totals) num.Amount {
	if t.Due != nil {
		return *t.Due
	}
	return t.Payable
}

// goblApplyDeclaredTotals records the sender's figures and tags the document so
// GOBL leaves them alone. Calculation stops under the tag, so every amount the
// document would otherwise derive is supplied here.
func goblApplyDeclaredTotals(in *Invoice, out *bill.Invoice) error {
	// Align the sender's precision with the currency's.
	exp := out.Currency.Def().Zero().Exp()
	declared := func(s string) (num.Amount, bool) {
		v, ok := goblDeclaredAmount(s)
		if !ok {
			return v, false
		}
		return v.RescaleUp(exp), true
	}

	for i, docLine := range in.Transaction.Lines {
		if i >= len(out.Lines) {
			break
		}
		line := out.Lines[i]
		if line == nil {
			continue
		}
		v, ok := goblDeclaredLineAmount(docLine)
		if !ok {
			continue
		}
		// The sum stays calculated: no term carries the amount before
		// allowances and charges.
		v = v.RescaleUp(exp)
		line.Total = &v
	}

	t := out.Totals
	if t == nil {
		t = new(bill.Totals)
		out.Totals = t
	}
	s := goblSummary(in)
	if s == nil {
		return out.Calculate()
	}
	if v, ok := declared(s.LineTotalAmount); ok {
		t.Sum = v
	}
	if v, ok := declared(s.TaxBasisTotalAmount); ok {
		t.Total = v
	}
	if v, ok := declared(s.GrandTotalAmount); ok {
		t.TotalWithTax = v
		t.Payable = v
	}
	if v, ok := declared(s.TotalPrepaidAmount); ok {
		t.Advances = &v
	}
	if v, ok := declared(s.RoundingAmount); ok {
		t.Rounding = &v
	}
	// BT-115 is what remains to pay, which GOBL keeps in Due after advances.
	if v, ok := declared(s.DuePayableAmount); ok {
		if t.Advances != nil {
			t.Due = &v
		} else {
			t.Payable = v
		}
	}
	if s.TaxTotalAmount != nil {
		if v, ok := declared(s.TaxTotalAmount.Amount); ok {
			t.Tax = v
			// The breakdown must come from the document too, or it
			// contradicts the total just set.
			t.Taxes = goblDeclaredTaxBreakdown(in, exp)
		}
	}
	if v, ok := declared(s.Discounts); ok {
		t.Discount = &v
	}
	if v, ok := declared(s.Charges); ok {
		t.Charge = &v
	}

	// Derived values freeze once the tag is set, so re-derive the dues against
	// the declared payable.
	if out.Payment != nil {
		out.Payment.Terms.CalculateDues(out.Currency.Def().Zero(), t.Payable)
	}

	out.SetTags(tax.TagBypass)
	return out.Calculate()
}

// goblApplyRounding carries BT-114 into the calculation. GOBL keeps a supplied
// rounding amount rather than deriving one.
func goblApplyRounding(in *Invoice, out *bill.Invoice) {
	s := goblSummary(in)
	if s == nil {
		return
	}
	v, ok := goblDeclaredAmount(s.RoundingAmount)
	if !ok || v.IsZero() {
		return
	}
	if out.Totals == nil {
		out.Totals = new(bill.Totals)
	}
	out.Totals.Rounding = &v
}

// goblDeclaredTaxBreakdown rebuilds BG-23 from the document's own trade tax
// entries, so it agrees with the tax total recorded alongside it.
func goblDeclaredTaxBreakdown(in *Invoice, exp uint32) *tax.Total {
	if in.Transaction.Settlement == nil {
		return nil
	}
	total := new(tax.Total)
	for _, tt := range in.Transaction.Settlement.Tax {
		base, ok := goblDeclaredAmount(tt.BasisAmount)
		if !ok {
			continue
		}
		amount, ok := goblDeclaredAmount(tt.CalculatedAmount)
		if !ok {
			continue
		}
		rate := &tax.RateTotal{
			Base:   base.RescaleUp(exp),
			Amount: amount.RescaleUp(exp),
		}
		ext := make(cbc.CodeMap)
		if tt.CategoryCode != "" {
			ext[untdid.ExtKeyTaxCategory] = cbc.Code(tt.CategoryCode)
		}
		if tt.ExemptionReasonCode != "" {
			ext[cef.ExtKeyVATEX] = cbc.Code(tt.ExemptionReasonCode)
		}
		if len(ext) > 0 {
			rate.Ext = tax.ExtensionsOf(ext)
		}
		if tt.RateApplicablePercent != "" {
			p, err := num.PercentageFromString(strings.TrimSuffix(tt.RateApplicablePercent, "%") + "%")
			if err == nil {
				rate.Percent = &p
			}
		}
		cat := goblCategoryTotal(total, cbc.Code(tt.TypeCode))
		cat.Rates = append(cat.Rates, rate)
		cat.Amount = cat.Amount.MatchPrecision(rate.Amount).Add(rate.Amount)
		total.Sum = total.Sum.MatchPrecision(rate.Amount).Add(rate.Amount)
	}
	if len(total.Categories) == 0 {
		return nil
	}
	return total
}

// goblSummary returns the document's header monetary summation, if it has one.
func goblSummary(in *Invoice) *Summary {
	if in.Transaction == nil || in.Transaction.Settlement == nil {
		return nil
	}
	return in.Transaction.Settlement.Summary
}

// goblDeclaredLineAmount reads a line's declared net amount (BT-131).
func goblDeclaredLineAmount(l *Line) (num.Amount, bool) {
	if l == nil || l.TradeSettlement == nil || l.TradeSettlement.Sum == nil {
		return num.AmountZero, false
	}
	return goblDeclaredAmount(l.TradeSettlement.Sum.Amount)
}

// goblDeclaredAmount parses a declared amount, reporting whether one was there.
func goblDeclaredAmount(s string) (num.Amount, bool) {
	if s == "" {
		return num.AmountZero, false
	}
	v, err := num.AmountFromString(s)
	if err != nil {
		return num.AmountZero, false
	}
	return v, true
}

// goblCategoryTotal finds or adds a category's running total. Each declared
// entry is a rate within its category, not a category of its own.
func goblCategoryTotal(total *tax.Total, code cbc.Code) *tax.CategoryTotal {
	for _, c := range total.Categories {
		if c.Code == code {
			return c
		}
	}
	cat := &tax.CategoryTotal{Code: code}
	total.Categories = append(total.Categories, cat)
	return cat
}
