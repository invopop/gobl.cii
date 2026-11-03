package cii

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func summedLine(sum string) *Line {
	return &Line{
		LineDoc:         &LineDoc{ID: "1"},
		TradeSettlement: &TradeSettlement{Sum: &Summation{Amount: sum}},
	}
}

func TestReconcileDefensive(t *testing.T) {
	total := num.MakeAmount(1000, 2)
	price := num.MakeAmount(1000, 2)

	t.Run("unreconciled groups skip missing lines", func(t *testing.T) {
		srcs := []lineSource{
			{doc: summedLine("10.00"), children: []*Line{summedLine("10.00")}},
			{doc: summedLine("10.00"), children: []*Line{summedLine("10.00")}},
		}
		out := &bill.Invoice{Lines: []*bill.Line{nil}}
		assert.Nil(t, goblUnreconciledGroups(out, srcs))
	})

	t.Run("base quantities", func(t *testing.T) {
		srcs := []lineSource{
			{doc: summedLine("20.00")},
			{doc: summedLine("10.00")},
			{doc: summedLine("99.00")},
			{doc: &Line{}},
		}
		line := func(per *num.Amount) *bill.Line {
			p := price
			tt := total
			return &bill.Line{Item: &org.Item{Price: &p, Per: per}, Total: &tt}
		}
		two := num.MakeAmount(2, 0)
		out := &bill.Invoice{Lines: []*bill.Line{nil, line(&two), line(&two), line(&two)}}
		swaps := goblDropConflictingBaseQuantities(srcs, out)
		require.Len(t, swaps, 1)
		assert.Equal(t, 2, swaps[0].index)
		assert.Nil(t, out.Lines[2].Item.Per)

		swaps = append(swaps, priceSwap{index: 9}, priceSwap{index: 0})
		assert.True(t, goblRestoreBaseQuantities(srcs, out, swaps))
		assert.Equal(t, "2", out.Lines[2].Item.Per.String())
	})

	t.Run("declared totals with missing lines", func(t *testing.T) {
		srcs := []lineSource{{doc: summedLine("10.00")}}
		assert.False(t, goblDeclaredTotalsAgree(&Invoice{}, &bill.Invoice{}, srcs))
		assert.False(t, goblDeclaredTotalsAgree(&Invoice{}, &bill.Invoice{}, nil))
	})

	t.Run("apply declared totals without a summary", func(t *testing.T) {
		out := &bill.Invoice{Currency: currency.EUR, Lines: []*bill.Line{nil}}
		srcs := []lineSource{{doc: summedLine("10.00")}, {doc: summedLine("10.00")}}
		assert.NotPanics(t, func() {
			_ = goblApplyDeclaredTotals(&Invoice{}, out, srcs)
		})
	})

	t.Run("declared amounts", func(t *testing.T) {
		_, ok := goblDeclaredAmount("bad")
		assert.False(t, ok)
		_, ok = goblDeclaredLineAmount(nil)
		assert.False(t, ok)
		assert.Nil(t, goblDeclaredTaxBreakdown(&Invoice{Transaction: &Transaction{}}, 2))
	})
}
