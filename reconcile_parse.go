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

// goblReconcileTotals aligns the converted document with the figures the sender
// declared, preferring the terms EN 16931 actually makes binding.
//
// The invoice line net amount (BT-131) is mandatory (BR-24) and is the only
// line figure the document totals are summed from (BR-CO-10, BR-CO-13). The
// item price base quantity (BT-149) is optional and appears in no rule at all,
// and GOBL has nowhere to store it: it can only be folded into the unit price.
// So when a line's declared amount and its base quantity disagree, the declared
// amount decides and the base quantity is dropped.
//
// A document that still does not reconcile is one whose arithmetic we cannot
// reproduce from the terms it carries. Rather than store our own figures in
// place of the sender's, it is tagged for bypass and the declared amounts are
// recorded verbatim.
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

// goblDropConflictingBaseQuantities removes the base quantity from any line
// whose total only matches the declared amount (BT-131) without it. Lines that
// match either way keep it, since dividing is the reading the standard
// describes.
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
		// The line disagrees with its declared amount. Retry it with the base
		// quantity left out, which is the only other reading of the price the
		// document supports.
		price, err := num.AmountFromString(np.Amount)
		if err != nil {
			continue
		}
		swaps = append(swaps, priceSwap{index: i, price: *line.Item.Price})
		line.Item.Price = &price
	}
	return swaps
}

// goblRestoreBaseQuantities puts back the standard reading of the price on any
// swapped line that still does not match its declared amount.
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

// goblDeclaredTotalsAgree reports whether every declared amount in the document
// is reproduced by the calculated one: the line amounts (BT-131), the sum of
// them (BT-106), the total without VAT (BT-109), the VAT total (BT-110), the
// total with VAT (BT-112) and the amount due for payment (BT-115).
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

// goblPayableAmount returns the total the amount due for payment (BT-115) is
// built from. GOBL keeps the amount still owed in Due once advances are
// deducted, and only falls back to Payable when there are none — the same
// choice the outbound mapping makes.
func goblPayableAmount(t *bill.Totals) num.Amount {
	if t.Due != nil {
		return *t.Due
	}
	return t.Payable
}

// goblApplyDeclaredTotals records the sender's own figures and tags the document
// so GOBL leaves them alone. Under the bypass tag calculation stops before any
// total is derived, so every amount has to be supplied here.
func goblApplyDeclaredTotals(in *Invoice, out *bill.Invoice) error {
	// Declared amounts carry whatever precision the sender wrote them with, so
	// align them with the currency before they become the document's own.
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
		// The sum stays as calculated: no business term carries the line amount
		// before allowances and charges, so the sender's own total is all we can
		// state with authority.
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
	// BT-115 is what remains to be paid, which GOBL keeps in Due whenever
	// advances have been deducted.
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
			// The breakdown has to come from the document too. Leaving the
			// calculated one in place would contradict the total just set.
			t.Taxes = goblDeclaredTaxBreakdown(in, exp)
		}
	}
	if v, ok := declared(s.Discounts); ok {
		t.Discount = &v
	}
	if v, ok := declared(s.Charges); ok {
		t.Charge = &v
	}

	// Anything GOBL derives from the totals is frozen once the tag is set, so
	// the payment dues have to be re-derived against the declared payable
	// rather than the one calculated before it was replaced.
	if out.Payment != nil {
		out.Payment.Terms.CalculateDues(out.Currency.Def().Zero(), t.Payable)
	}

	out.SetTags(tax.TagBypass)
	return out.Calculate()
}

// goblApplyRounding carries the rounding amount the sender applied to the amount
// due (BT-114) into the calculation. GOBL keeps a rounding amount that was
// supplied rather than deriving one, so setting it before calculating lets the
// amount due come out as the sender stated it.
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

// goblDeclaredTaxBreakdown rebuilds the VAT breakdown (BG-23) from the
// document's own trade tax entries, so the categories agree with the tax total
// recorded alongside them.
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

// goblDeclaredAmount parses a declared monetary amount, reporting whether one
// was present at all.
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

// goblCategoryTotal finds the running total for a tax category, adding one if
// the category has not been seen yet. Each declared tax entry is a rate within
// its category, not a category of its own.
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
