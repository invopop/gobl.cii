package cii

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
)

const testCurrencyEUR = "EUR"

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
