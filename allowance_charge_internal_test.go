package cii

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pct(t *testing.T, s string) *num.Percentage {
	t.Helper()
	p, err := num.PercentageFromString(s)
	require.NoError(t, err)
	return &p
}

func TestNewChargeAndDiscount(t *testing.T) {
	amount := num.MakeAmount(1050, 2)
	base := num.MakeAmount(10000, 2)

	t.Run("a charge carries the charge reason code", func(t *testing.T) {
		ac := newCharge(&bill.Charge{
			Amount:  amount,
			Base:    &base,
			Reason:  "Freight",
			Percent: pct(t, "10.5%"),
			Ext:     tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyCharge: "FC"}),
		}, testCurrencyEUR)

		assert.True(t, ac.ChargeIndicator.Value)
		assert.Equal(t, "10.50", ac.Amount)
		assert.Equal(t, "100.00", ac.Base)
		assert.Equal(t, "Freight", ac.Reason)
		assert.Equal(t, "FC", ac.ReasonCode)
		assert.Equal(t, "10.5", ac.Percent)
	})

	t.Run("a discount carries the allowance reason code", func(t *testing.T) {
		ac := newDiscount(&bill.Discount{
			Amount: amount,
			Reason: "Loyalty",
			Ext:    tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyAllowance: "95"}),
		}, testCurrencyEUR)

		assert.False(t, ac.ChargeIndicator.Value)
		assert.Equal(t, "10.50", ac.Amount)
		assert.Empty(t, ac.Base, "no base was given")
		assert.Equal(t, "95", ac.ReasonCode)
		assert.Empty(t, ac.Percent)
	})

	t.Run("amounts follow the currency, not a fixed two decimals", func(t *testing.T) {
		ac := newCharge(&bill.Charge{Amount: num.MakeAmount(105000, 2)}, "JPY")
		assert.Equal(t, "1050", ac.Amount)
	})

	t.Run("taxes come through as the category", func(t *testing.T) {
		ac := newCharge(&bill.Charge{
			Amount: amount,
			Taxes:  tax.Set{{Category: testCategoryVAT, Percent: pct(t, "21%")}},
		}, testCurrencyEUR)
		require.NotNil(t, ac.Tax)
	})

	t.Run("a bare charge and discount", func(t *testing.T) {
		c := newCharge(&bill.Charge{Amount: amount}, testCurrencyEUR)
		assert.Empty(t, c.Reason)
		assert.Empty(t, c.ReasonCode)
		assert.Nil(t, c.Tax)

		d := newDiscount(&bill.Discount{Amount: amount}, testCurrencyEUR)
		assert.Empty(t, d.Reason)
		assert.Nil(t, d.Tax)
	})
}

func TestMakeLineChargeAndDiscount(t *testing.T) {
	amount := num.MakeAmount(250, 2)
	base := num.MakeAmount(5000, 2)

	t.Run("a line charge with a percentage keeps its base", func(t *testing.T) {
		ac := makeLineCharge(&bill.LineCharge{
			Amount:  amount,
			Base:    &base,
			Percent: pct(t, "5%"),
			Reason:  "Handling",
			Ext:     tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyCharge: "FC"}),
		}, testCurrencyEUR)

		assert.True(t, ac.ChargeIndicator.Value)
		assert.Equal(t, testAmountSmall, ac.Amount)
		assert.Equal(t, testAmountHalf, ac.Base)
		assert.Equal(t, "5", ac.Percent)
		assert.Equal(t, "Handling", ac.Reason)
		assert.Equal(t, "FC", ac.ReasonCode)
	})

	t.Run("a percentage without a base carries neither", func(t *testing.T) {
		// BT-138 only makes sense alongside BT-137, so the percentage is
		// dropped when the base is missing.
		ac := makeLineCharge(&bill.LineCharge{Amount: amount, Percent: pct(t, "5%")}, testCurrencyEUR)
		assert.Empty(t, ac.Percent)
		assert.Empty(t, ac.Base)
	})

	t.Run("a line discount", func(t *testing.T) {
		ac := makeLineDiscount(&bill.LineDiscount{
			Amount:  amount,
			Base:    &base,
			Percent: pct(t, "5%"),
			Reason:  "Bulk",
			Ext:     tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyAllowance: "95"}),
		}, testCurrencyEUR)

		assert.False(t, ac.ChargeIndicator.Value)
		assert.Equal(t, testAmountSmall, ac.Amount)
		assert.Equal(t, testAmountHalf, ac.Base)
		assert.Equal(t, "95", ac.ReasonCode)
		assert.Equal(t, "Bulk", ac.Reason)
	})

	t.Run("a bare line discount", func(t *testing.T) {
		ac := makeLineDiscount(&bill.LineDiscount{Amount: amount}, testCurrencyEUR)
		assert.Empty(t, ac.Reason)
		assert.Empty(t, ac.Percent)
		assert.Empty(t, ac.Base)
	})
}

func TestGoblNewLineChargeAndDiscount(t *testing.T) {
	t.Run("a charge maps back", func(t *testing.T) {
		c, err := goblNewLineCharge(&AllowanceCharge{
			Amount:     testAmountSmall,
			Reason:     "Handling",
			ReasonCode: "FC",
			Percent:    "5",
		})
		require.NoError(t, err)
		assert.Equal(t, testAmountSmall, c.Amount.String())
		assert.Equal(t, "Handling", c.Reason)
		assert.Equal(t, "FC", c.Ext.Get(untdid.ExtKeyCharge).String())
		require.NotNil(t, c.Percent)
		assert.Equal(t, "5%", c.Percent.String())
	})

	t.Run("a percentage already carrying its sign is not doubled", func(t *testing.T) {
		c, err := goblNewLineCharge(&AllowanceCharge{Amount: testAmountSmall, Percent: "5%"})
		require.NoError(t, err)
		require.NotNil(t, c.Percent)
		assert.Equal(t, "5%", c.Percent.String())
	})

	t.Run("a discount maps back", func(t *testing.T) {
		d, err := goblNewLineDiscount(&AllowanceCharge{
			Amount:     testAmountSmall,
			Reason:     "Bulk",
			ReasonCode: "95",
			Percent:    "5",
		})
		require.NoError(t, err)
		assert.Equal(t, testAmountSmall, d.Amount.String())
		assert.Equal(t, "Bulk", d.Reason)
		assert.Equal(t, "95", d.Ext.Get(untdid.ExtKeyAllowance).String())
		require.NotNil(t, d.Percent)
	})

	t.Run("an amount that is not a number is an error", func(t *testing.T) {
		_, err := goblNewLineCharge(&AllowanceCharge{Amount: testNotANumber})
		assert.Error(t, err)

		_, err = goblNewLineDiscount(&AllowanceCharge{Amount: testNotANumber})
		assert.Error(t, err)
	})

	t.Run("a percentage that is not a number is an error", func(t *testing.T) {
		_, err := goblNewLineCharge(&AllowanceCharge{Amount: testAmountSmall, Percent: testNotAPercent})
		assert.Error(t, err)

		_, err = goblNewLineDiscount(&AllowanceCharge{Amount: testAmountSmall, Percent: testNotAPercent})
		assert.Error(t, err)
	})

	t.Run("the amount alone is enough", func(t *testing.T) {
		c, err := goblNewLineCharge(&AllowanceCharge{Amount: testAmountSmall})
		require.NoError(t, err)
		assert.Empty(t, c.Reason)
		assert.Nil(t, c.Percent)
		assert.True(t, c.Ext.IsZero())
	})
}
