package cii

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRescaleToCurrency(t *testing.T) {
	tests := []struct {
		name     string
		amount   num.Amount
		currency string
		expected string
	}{
		{"rounds down to the currency's precision", num.MakeAmount(67273, 4), testCurrencyEUR, "6.73"},
		{"pads up to the currency's precision", num.MakeAmount(6, 0), testCurrencyEUR, "6.00"},
		{"a currency without subunits", num.MakeAmount(12345, 2), "JPY", "123"},
		{"a three decimal currency", num.MakeAmount(12345, 2), "BHD", "123.450"},
		{"an unknown currency keeps the amount as it stands", num.MakeAmount(67273, 4), "ZZZ", "6.7273"},
		{"an empty currency keeps the amount as it stands", num.MakeAmount(67273, 4), "", "6.7273"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, rescaleToCurrency(tt.amount, tt.currency))
		})
	}
}

func TestLineCurrency(t *testing.T) {
	inv := &bill.Invoice{Currency: currency.EUR}

	t.Run("falls back to the document's currency", func(t *testing.T) {
		assert.Equal(t, testCurrencyEUR, lineCurrency(inv, &bill.Line{Item: &org.Item{}}))
	})

	t.Run("a line with no item at all", func(t *testing.T) {
		assert.Equal(t, testCurrencyEUR, lineCurrency(inv, &bill.Line{}))
	})

	t.Run("the item's currency wins when it sets one", func(t *testing.T) {
		l := &bill.Line{Item: &org.Item{Currency: currency.USD}}
		assert.Equal(t, "USD", lineCurrency(inv, l))
	})
}

func TestExceedsCurrencyPrecision(t *testing.T) {
	over := num.MakeAmount(67273, 4) // 6.7273, one decimal more than EUR allows
	fits := num.MakeAmount(673, 2)   // 6.73

	invoice := func(f func(inv *bill.Invoice)) *bill.Invoice {
		inv := &bill.Invoice{
			Currency: currency.EUR,
			Totals:   &bill.Totals{Sum: num.MakeAmount(0, 2)},
			Lines:    []*bill.Line{{Index: 1, Total: &fits, Sum: &fits}},
		}
		if f != nil {
			f(inv)
		}
		return inv
	}

	t.Run("no totals means nothing to round", func(t *testing.T) {
		inv := invoice(nil)
		inv.Totals = nil
		assert.False(t, exceedsCurrencyPrecision(inv))
	})

	t.Run("an unknown currency is left alone", func(t *testing.T) {
		assert.False(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Currency = currency.Code("ZZZ")
			inv.Lines[0].Total = &over
		})))
	})

	t.Run("amounts within the currency", func(t *testing.T) {
		assert.False(t, exceedsCurrencyPrecision(invoice(nil)))
	})

	cases := map[string]func(inv *bill.Invoice){
		"line total":             func(inv *bill.Invoice) { inv.Lines[0].Total = &over },
		"line sum":               func(inv *bill.Invoice) { inv.Lines[0].Sum = &over },
		"line discount":          func(inv *bill.Invoice) { inv.Lines[0].Discounts = []*bill.LineDiscount{{Amount: over}} },
		"line charge":            func(inv *bill.Invoice) { inv.Lines[0].Charges = []*bill.LineCharge{{Amount: over}} },
		"document discount":      func(inv *bill.Invoice) { inv.Discounts = []*bill.Discount{{Amount: over}} },
		"document discount base": func(inv *bill.Invoice) { inv.Discounts = []*bill.Discount{{Amount: fits, Base: &over}} },
		"document charge":        func(inv *bill.Invoice) { inv.Charges = []*bill.Charge{{Amount: over}} },
		"document charge base":   func(inv *bill.Invoice) { inv.Charges = []*bill.Charge{{Amount: fits, Base: &over}} },
		"tax rate base": func(inv *bill.Invoice) {
			inv.Totals.Taxes = &tax.Total{Categories: []*tax.CategoryTotal{
				{Rates: []*tax.RateTotal{{Base: over, Amount: fits}}},
			}}
		},
		"tax rate amount": func(inv *bill.Invoice) {
			inv.Totals.Taxes = &tax.Total{Categories: []*tax.CategoryTotal{
				{Rates: []*tax.RateTotal{{Base: fits, Amount: over}}},
			}}
		},
	}
	for name, apply := range cases {
		t.Run(name, func(t *testing.T) {
			assert.True(t, exceedsCurrencyPrecision(invoice(apply)))
		})
	}
}

// testInvoice builds the smallest invoice that calculates, priced so that the
// line total (3 × 10.555 = 31.665) needs one decimal more than EUR allows.
func testInvoice(t *testing.T) *bill.Invoice {
	t.Helper()

	price := num.MakeAmount(10555, 3)
	inv := &bill.Invoice{
		Regime:    tax.WithRegime("ES"),
		Code:      "TEST-1",
		IssueDate: cal.MakeDate(2024, 1, 15),
		Currency:  currency.EUR,
		Supplier:  &org.Party{Name: "Supplier", TaxID: &tax.Identity{Country: "ES", Code: "B98602642"}},
		Customer:  &org.Party{Name: "Customer", TaxID: &tax.Identity{Country: "ES", Code: "A39200019"}},
		Lines: []*bill.Line{{
			Quantity: num.MakeAmount(3, 0),
			Item:     &org.Item{Name: "Item", Price: &price},
			Taxes:    tax.Set{{Category: testCategoryVAT, Percent: num.NewPercentage(210, 3)}},
		}},
	}
	require.NoError(t, inv.Calculate())
	return inv
}

func TestRoundToCurrency(t *testing.T) {
	t.Run("an invoice that already fits is left untouched", func(t *testing.T) {
		inv := testInvoice(t)
		price := num.MakeAmount(1000, 2)
		inv.Lines[0].Item.Price = &price
		require.NoError(t, inv.Calculate())
		require.False(t, exceedsCurrencyPrecision(inv))

		before := inv.Totals.Payable
		require.NoError(t, roundToCurrency(inv))

		assert.Equal(t, before.String(), inv.Totals.Payable.String())
		assert.Nil(t, inv.Totals.Rounding, "no rounding should be invented")
	})

	t.Run("a document without a tax block gets the currency rule", func(t *testing.T) {
		inv := testInvoice(t)
		require.Nil(t, inv.Tax, "the fixture is expected to carry no tax block")
		require.Equal(t, "31.665", inv.Lines[0].Total.String())
		payable := inv.Totals.Payable

		require.NoError(t, roundToCurrency(inv))

		require.NotNil(t, inv.Tax)
		assert.Equal(t, tax.RoundingRuleCurrency, inv.Tax.Rounding)
		assert.Equal(t, "31.67", inv.Lines[0].Total.String(), "BT-131 must fit the currency")
		assert.Equal(t, payable.String(), inv.Totals.Payable.String(), "the amount owed must not move")
	})

	t.Run("an existing rounding total is added to, not replaced", func(t *testing.T) {
		inv := testInvoice(t)

		existing := num.MakeAmount(-5, 2)
		inv.Totals.Rounding = &existing
		require.NoError(t, inv.Calculate())
		payable := inv.Totals.Payable

		require.NoError(t, roundToCurrency(inv))

		require.NotNil(t, inv.Totals.Rounding)
		assert.Equal(t, payable.String(), inv.Totals.Payable.String(),
			"the amount owed must survive the rounding")
		// Rounding the line to 31.67 gains a cent, so the -0.05 the document
		// already carried becomes -0.06. Replacing it rather than adding to it
		// would read -0.01 and move the payable total by the 0.05 it dropped.
		assert.Equal(t, "-0.06", inv.Totals.Rounding.String())
	})

	t.Run("a recalculation that cannot resolve its taxes is reported", func(t *testing.T) {
		inv := testInvoice(t)
		// A rate key the regime does not define: the recalculation fails, and
		// the failure must surface rather than yield a half-rounded document.
		inv.Lines[0].Taxes = tax.Set{{Category: testCategoryVAT, Rate: "nonsense"}}

		assert.Error(t, roundToCurrency(inv))
	})
}
