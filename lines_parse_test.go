package cii_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseXMLLines function
func TestParseCtoGLines(t *testing.T) {
	// Basic Invoice 1
	t.Run("invoice-test-01.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "invoice-test-01.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		require.Len(t, lines, 2)
		priceLine1, _ := num.AmountFromString("5350.00")
		priceLine2, _ := num.AmountFromString("149.00")

		assert.Equal(t, "2h Beschaffung + Aufbau des neuen Tisches a 25€/h netto + 7% MwSt.", lines[0].Item.Name)
		assert.Equal(t, priceLine1, *lines[0].Item.Price)
		assert.Equal(t, num.MakeAmount(1, 0), lines[0].Quantity)
		assert.Equal(t, "VAT", string(lines[0].Taxes[0].Category))
		percent, err := num.PercentageFromString("7%")
		require.NoError(t, err)
		assert.Equal(t, &percent, lines[0].Taxes[0].Percent)

		assert.Equal(t, "1x Couchtisch inklusive 19% MwSt.", lines[1].Item.Name)
		assert.Equal(t, priceLine2, *lines[1].Item.Price)
		assert.Equal(t, num.MakeAmount(1, 0), lines[1].Quantity)
		assert.Equal(t, "VAT", string(lines[1].Taxes[0].Category))
		percent, err = num.PercentageFromString("19%")
		require.NoError(t, err)
		assert.Equal(t, &percent, lines[1].Taxes[0].Percent)

	})

	//Basic Invoice 2
	t.Run("CII_example1.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_example1.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		require.Len(t, lines, 20)

		assert.Equal(t, "PATAT FRITES 10MM 10KG", lines[0].Item.Name)
		assert.Equal(t, num.MakeAmount(995, 2), *lines[0].Item.Price)
		assert.Equal(t, org.Unit("piece"), lines[0].Item.Unit)
		assert.Equal(t, num.MakeAmount(2, 0), lines[0].Quantity)
		assert.Equal(t, "VAT", string(lines[0].Taxes[0].Category))
		percent, err := num.PercentageFromString("6%")
		require.NoError(t, err)
		assert.Equal(t, &percent, lines[0].Taxes[0].Percent)

		assert.Equal(t, "KAAS 50PL. JONG BEL. 1KG", lines[1].Item.Name)
		assert.Equal(t, num.MakeAmount(985, 2), *lines[1].Item.Price)
		assert.Equal(t, org.Unit("piece"), lines[1].Item.Unit)
		assert.Equal(t, num.MakeAmount(1, 0), lines[1].Quantity)
		assert.Equal(t, "VAT", string(lines[1].Taxes[0].Category))
		percent, err = num.PercentageFromString("6%")
		require.NoError(t, err)
		assert.Equal(t, &percent, lines[1].Taxes[0].Percent)

		assert.Equal(t, "POT KETCHUP 3 LT", lines[2].Item.Name)
		assert.Equal(t, num.MakeAmount(829, 2), *lines[2].Item.Price)
		assert.Equal(t, org.Unit("piece"), lines[2].Item.Unit)
		assert.Equal(t, num.MakeAmount(1, 0), lines[2].Quantity)
		assert.Equal(t, "VAT", string(lines[2].Taxes[0].Category))
		percent, err = num.PercentageFromString("6%")
		require.NoError(t, err)
		assert.Equal(t, &percent, lines[2].Taxes[0].Percent)

	})

	// Invoice with Description and Origin Country
	t.Run("CII_example2.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		require.NotEmpty(t, lines)

		assert.Equal(t, "Laptop computer", lines[0].Item.Name)
		assert.Equal(t, "Processor: Intel Core 2 Duo SU9400 LV (1.4GHz). RAM: 3MB. Screen 1440x900", lines[0].Item.Description)
		assert.Equal(t, l10n.ISOCountryCode("DE"), lines[0].Item.Origin)
		assert.Equal(t, cbc.Code("JB007"), lines[0].Item.Ref)
		assert.Equal(t, "1234567890128", lines[0].Item.Identities[0].Code.String())
		assert.Equal(t, "0088", lines[0].Item.Identities[0].Ext[iso.ExtKeySchemeID].String())
	})

}
