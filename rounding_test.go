package cii_test

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurrencyRounding covers the EN 16931 BR-DEC-* rules, which cap nearly
// every monetary amount at the currency's precision. GOBL's `precise` rounding
// rule — the default for every regime but Greece, and what RemoveIncludedTaxes
// switches a document to since GOBL v0.505 — keeps line amounts at a higher
// precision, so the converter has to round the document before writing it out.
func TestCurrencyRounding(t *testing.T) {
	t.Run("prices include tax", func(t *testing.T) {
		env := loadEnvelope(t, filepath.Join("en16931", "invoice-prices-include.json"))
		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextEN16931V2017))
		require.NoError(t, err)

		s := out.Transaction.Settlement.Summary
		require.NotNil(t, s)

		// The rounded lines add up to the document total (BR-CO-10), and the
		// cents the rounding gained are declared as BT-114 so that the payable
		// amount still matches the tax-inclusive original.
		assert.Equal(t, "231.11", s.LineTotalAmount)
		assert.Equal(t, "-0.02", s.RoundingAmount)
		assert.Equal(t, "275.02", s.GrandTotalAmount)
		assert.Equal(t, "275.00", s.DuePayableAmount)

		assertAmountsFitCurrency(t, out)
		assertTotalsAddUp(t, out)
	})

	t.Run("line total with more decimals than the currency", func(t *testing.T) {
		env := loadEnvelope(t, filepath.Join("en16931", "invoice-complete.json"))
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// 3 × 10.555 = 31.665, one decimal more than EUR allows.
		price := num.MakeAmount(10555, 3)
		inv.Lines = inv.Lines[:1]
		inv.Lines[0].Quantity = num.MakeAmount(3, 0)
		inv.Lines[0].Item.Price = &price
		inv.Lines[0].Discounts = nil
		inv.Lines[0].Charges = nil
		require.NoError(t, env.Calculate())
		require.Equal(t, "31.665", inv.Lines[0].Total.String())

		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextEN16931V2017))
		require.NoError(t, err)

		require.Len(t, out.Transaction.Lines, 1)
		assert.Equal(t, "31.67", out.Transaction.Lines[0].TradeSettlement.Sum.Amount,
			"BT-131 must fit the currency")
		assert.Equal(t, "31.67", out.Transaction.Settlement.Summary.LineTotalAmount,
			"the document total must agree with the rounded line")
		assertAmountsFitCurrency(t, out)
		assertTotalsAddUp(t, out)
	})

	t.Run("many lines keep the payable amount exact", func(t *testing.T) {
		for _, lines := range []int{1, 10, 50, 200} {
			t.Run(strconv.Itoa(lines)+" lines", func(t *testing.T) {
				env := loadEnvelope(t, filepath.Join("en16931", "invoice-prices-include.json"))
				inv, ok := env.Extract().(*bill.Invoice)
				require.True(t, ok)

				// 9.99 including VAT is never an exact number of cents net.
				base := inv.Lines[0]
				inv.Lines = make([]*bill.Line, 0, lines)
				for range lines {
					l, item := *base, *base.Item
					price := num.MakeAmount(999, 2)
					item.Price = &price
					l.Item = &item
					l.Quantity = num.MakeAmount(1, 0)
					l.Discounts, l.Charges = nil, nil
					inv.Lines = append(inv.Lines, &l)
				}
				require.NoError(t, env.Calculate())
				payable := inv.Totals.Payable

				out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextEN16931V2017))
				require.NoError(t, err)

				assertAmountsFitCurrency(t, out)
				assertTotalsAddUp(t, out)
				assert.Equal(t, payable.String(), out.Transaction.Settlement.Summary.DuePayableAmount,
					"the amount owed must not move, however the cents fall")
			})
		}
	})

	t.Run("a currency without subunits", func(t *testing.T) {
		env := loadEnvelope(t, filepath.Join("en16931", "invoice-complete.json"))
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// JPY has no subunits, so hardcoding two decimals would be wrong.
		inv.Currency = "JPY"
		inv.Lines = inv.Lines[:1]
		inv.Lines[0].Discounts = nil
		inv.Lines[0].Charges = nil
		price := num.MakeAmount(1000, 0)
		inv.Lines[0].Item.Price = &price
		inv.Lines[0].Item.Currency = ""
		require.NoError(t, env.Calculate())

		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextEN16931V2017))
		require.NoError(t, err)

		s := out.Transaction.Settlement.Summary
		assert.NotContains(t, s.LineTotalAmount, ".",
			"a currency without subunits must carry no decimals, got %q", s.LineTotalAmount)
		assert.NotContains(t, out.Transaction.Lines[0].TradeSettlement.Sum.Amount, ".")
	})
}

// assertAmountsFitCurrency checks every amount the BR-DEC-* rules constrain to
// two decimals. Unit prices (BT-146, BT-148) are exempt and so not included.
func assertAmountsFitCurrency(t *testing.T, out *cii.Invoice) {
	t.Helper()

	check := func(field, value string) {
		if value == "" {
			return
		}
		if _, frac, found := strings.Cut(value, "."); found {
			assert.LessOrEqual(t, len(frac), 2, "%s has more than 2 decimals: %s", field, value)
		}
	}

	stlm := out.Transaction.Settlement
	if s := stlm.Summary; s != nil {
		check("BT-106 LineTotalAmount", s.LineTotalAmount)
		check("BT-107 AllowanceTotalAmount", s.Discounts)
		check("BT-108 ChargeTotalAmount", s.Charges)
		check("BT-109 TaxBasisTotalAmount", s.TaxBasisTotalAmount)
		check("BT-112 GrandTotalAmount", s.GrandTotalAmount)
		check("BT-113 TotalPrepaidAmount", s.TotalPrepaidAmount)
		check("BT-114 RoundingAmount", s.RoundingAmount)
		check("BT-115 DuePayableAmount", s.DuePayableAmount)
		if s.TaxTotalAmount != nil {
			check("BT-110 TaxTotalAmount", s.TaxTotalAmount.Amount)
		}
	}
	for _, tx := range stlm.Tax {
		check("BT-116 BasisAmount", tx.BasisAmount)
		check("BT-117 CalculatedAmount", tx.CalculatedAmount)
	}
	for _, ac := range stlm.AllowanceCharges {
		check("BT-92/99 Amount", ac.Amount)
		check("BT-93/100 BaseAmount", ac.Base)
	}
	for _, l := range out.Transaction.Lines {
		if l.TradeSettlement != nil && l.TradeSettlement.Sum != nil {
			check("BT-131 LineTotalAmount", l.TradeSettlement.Sum.Amount)
		}
		if l.TradeSettlement != nil {
			for _, ac := range l.TradeSettlement.AllowanceCharge {
				check("BT-136/141 Amount", ac.Amount)
				check("BT-137/142 BaseAmount", ac.Base)
			}
		}
	}
}

// assertTotalsAddUp checks BR-CO-10, BR-CO-13, BR-CO-15 and BR-CO-16, the
// arithmetic the document totals have to satisfy. They are what makes rounding
// a document-wide decision rather than a per-field one: once a line net amount
// is rounded to the currency, the sums that derive from it have no freedom
// left.
func assertTotalsAddUp(t *testing.T, out *cii.Invoice) {
	t.Helper()

	amount := func(s string) num.Amount {
		if s == "" {
			return num.MakeAmount(0, 2)
		}
		a, err := num.AmountFromString(s)
		require.NoError(t, err)
		return a
	}

	s := out.Transaction.Settlement.Summary
	require.NotNil(t, s)

	// BR-CO-10: the sum of line net amounts is the sum of BT-131.
	sum := num.MakeAmount(0, 2)
	for _, l := range out.Transaction.Lines {
		if l.TradeSettlement != nil && l.TradeSettlement.Sum != nil {
			sum = sum.Add(amount(l.TradeSettlement.Sum.Amount))
		}
	}
	assert.Equal(t, sum.String(), s.LineTotalAmount, "BR-CO-10")

	// BR-CO-13: BT-109 = BT-106 - BT-107 + BT-108.
	basis := amount(s.LineTotalAmount).Subtract(amount(s.Discounts)).Add(amount(s.Charges))
	assert.Equal(t, basis.String(), s.TaxBasisTotalAmount, "BR-CO-13")

	// BR-CO-15: BT-112 = BT-109 + BT-110.
	var tax num.Amount
	if s.TaxTotalAmount != nil {
		tax = amount(s.TaxTotalAmount.Amount)
	}
	assert.Equal(t, amount(s.TaxBasisTotalAmount).Add(tax).String(), s.GrandTotalAmount, "BR-CO-15")

	// BR-CO-16: BT-115 = BT-112 - BT-113 + BT-114.
	due := amount(s.GrandTotalAmount).
		Subtract(amount(s.TotalPrepaidAmount)).
		Add(amount(s.RoundingAmount))
	assert.Equal(t, due.String(), s.DuePayableAmount, "BR-CO-16")
}
