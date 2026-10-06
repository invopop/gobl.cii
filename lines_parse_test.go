package cii_test

import (
	"os"
	"strings"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
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
		assert.Equal(t, org.UnitPiece, lines[0].Item.Unit)
		assert.Equal(t, num.MakeAmount(2, 0), lines[0].Quantity)
		assert.Equal(t, "VAT", string(lines[0].Taxes[0].Category))
		percent, err := num.PercentageFromString("6%")
		require.NoError(t, err)
		assert.Equal(t, &percent, lines[0].Taxes[0].Percent)

		assert.Equal(t, "KAAS 50PL. JONG BEL. 1KG", lines[1].Item.Name)
		assert.Equal(t, num.MakeAmount(985, 2), *lines[1].Item.Price)
		assert.Equal(t, org.UnitPiece, lines[1].Item.Unit)
		assert.Equal(t, num.MakeAmount(1, 0), lines[1].Quantity)
		assert.Equal(t, "VAT", string(lines[1].Taxes[0].Category))
		percent, err = num.PercentageFromString("6%")
		require.NoError(t, err)
		assert.Equal(t, &percent, lines[1].Taxes[0].Percent)

		assert.Equal(t, "POT KETCHUP 3 LT", lines[2].Item.Name)
		assert.Equal(t, num.MakeAmount(829, 2), *lines[2].Item.Price)
		assert.Equal(t, org.UnitPiece, lines[2].Item.Unit)
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
		assert.Equal(t, "0088", lines[0].Item.Identities[0].Ext.Get(iso.ExtKeySchemeID).String())

		// BG-32: Item attributes
		require.Len(t, lines[0].Item.Attributes, 1)
		assert.Equal(t, attrLabelColor, lines[0].Item.Attributes[0].Label)
		assert.Equal(t, attrValueBlack, lines[0].Item.Attributes[0].Text)

		// BT-158: Item classification
		classID := lines[0].Item.Identities[1]
		assert.Equal(t, cbc.Code("65434568"), classID.Code)
		assert.Equal(t, "STI", classID.Label)

		// BT-132: Purchase order line reference
		assert.Equal(t, cbc.Code("1"), lines[0].Order)

		// BT-133: Line buyer accounting reference
		assert.Equal(t, cbc.Code("BookingCode001"), lines[0].Cost)

		// BT-134/BT-135: Invoice line period
		require.NotNil(t, lines[0].Period, "Line period should not be nil")
		assert.Equal(t, "2013-06-01", lines[0].Period.Start.String())
		assert.Equal(t, "2013-06-01", lines[0].Period.End.String())
	})

	// Invoice with BasisQuantity
	t.Run("CII_example8.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_example8.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		require.NotEmpty(t, lines)

		// The basis quantity (BT-149) repeats the price, which no reading of
		// the standard supports: dividing by it gives a line of 16000.00
		// against a declared amount (BT-131) of 140.80. The declared amount
		// is the term EN 16931 makes binding, so the basis quantity is
		// dropped and the price stands as sent.
		assert.Equal(t, "0.00880", lines[0].Item.Price.String())
		assert.Equal(t, "140.80", lines[0].Total.String())
	})
}

// TestParseSubInvoiceLines covers the sub-invoice lines of the extended
// profile. Only DETAIL lines count towards the totals: a GROUP line restates
// the sum of its DETAIL lines and an INFORMATION line carries no amount. A
// GROUP line with no net price and an INFORMATION line with no trade agreement
// used to panic.
func TestParseSubInvoiceLines(t *testing.T) {
	e, err := parseInvoiceFrom(t, "sub-invoice-lines.xml")
	require.NoError(t, err)

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.Len(t, inv.Lines, 5)

	// The first group mixes tax rates, which a breakdown cannot, so it is read
	// flat with the GROUP line at a zero price.
	names := []string{"Kit", "Part A", "Part B", "Service bundle", "Assembly instructions"}
	totals := []string{rateZero, "100.00", "50.00", "20.00", rateZero}
	for i, l := range inv.Lines {
		assert.Equal(t, names[i], l.Item.Name)
		require.NotNil(t, l.Total)
		assert.Equal(t, totals[i], l.Total.String(), "line %d", i+1)
	}
	// The group's tax entry only names a due date: there is no tax to apply.
	assert.Empty(t, inv.Lines[0].Taxes)
	assert.Empty(t, inv.Lines[0].Breakdown)

	// The second shares one rate and becomes a breakdown.
	bundle := inv.Lines[3]
	require.Len(t, bundle.Breakdown, 1)
	assert.Equal(t, "Installation", bundle.Breakdown[0].Item.Name)
	require.Len(t, bundle.Taxes, 1)
	assert.Equal(t, "20.00%", bundle.Taxes[0].Percent.String())

	// The calculation reproduces the declared totals without being bypassed.
	assert.False(t, inv.HasTags(tax.TagBypass))
	assert.Equal(t, "170.00", inv.Totals.Sum.String())
	assert.Equal(t, "170.00", inv.Totals.Total.String())
	assert.Equal(t, "29.00", inv.Totals.Tax.String())
	assert.Equal(t, "199.00", inv.Totals.Payable.String())

	require.NoError(t, e.Validate())
}

// TestParseSubInvoiceLinesFlat covers the groups a breakdown cannot hold, which
// are read flat, next to the ones it can.
func TestParseSubInvoiceLinesFlat(t *testing.T) {
	e, err := parseInvoiceFrom(t, "sub-invoice-lines-flat.xml")
	require.NoError(t, err)

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)

	want := []struct {
		name      string
		total     string
		breakdown int
	}{
		// Nested groups.
		{"Display", rateZero, 0},
		{"Roast", "30.00", 0},
		{"Bundle", rateZero, 0},
		{"Colombia", "90.00", 0},
		// DETAIL lines ahead of their GROUP line still fold into it.
		{"Hardware", "550.00", 2},
		// A quantity that does not divide by the GROUP line's.
		{"Odd lot", rateZero, 0},
		{"Thing", "10.00", 0},
		// A GROUP line declaring more than its DETAIL lines add up to.
		{"Mismatch", rateZero, 0},
		{"Part", "40.00", 0},
		// INFORMATION lines describe a line that keeps its own price.
		{"Safety kit", "450.00", 2},
	}
	require.Len(t, inv.Lines, len(want))
	for i, w := range want {
		l := inv.Lines[i]
		assert.Equal(t, w.name, l.Item.Name, "line %d", i+1)
		assert.Equal(t, w.total, l.Total.String(), "line %d", i+1)
		assert.Len(t, l.Breakdown, w.breakdown, "line %d", i+1)
	}

	hardware := inv.Lines[4]
	assert.Equal(t, "Laser printer", hardware.Breakdown[0].Item.Name)
	assert.Equal(t, "-1", hardware.Breakdown[1].Quantity.String())

	kit := inv.Lines[9]
	assert.Equal(t, "45.00", kit.Item.Price.String())
	for _, sl := range kit.Breakdown {
		assert.Nil(t, sl.Item.Price)
		assert.Equal(t, "1", sl.Quantity.String())
	}

	assert.False(t, inv.HasTags(tax.TagBypass))
	assert.Equal(t, "1170.00", inv.Totals.Sum.String())
	assert.Equal(t, "234.00", inv.Totals.Tax.String())
	assert.Equal(t, "1404.00", inv.Totals.Payable.String())
	require.NoError(t, e.Validate())
}

// TestParseSubInvoiceLinesUnreconciled covers a group whose DETAIL lines
// declare amounts their prices do not produce: the breakdown would not come to
// the declared amount, so the group is read flat.
func TestParseSubInvoiceLinesUnreconciled(t *testing.T) {
	data, err := os.ReadFile(dataPath(pathParse, "sub-invoice-lines.xml"))
	require.NoError(t, err)
	// The bundle's only DETAIL line declares 25.00 at a 20.00 price.
	xml := strings.Replace(string(data), "<ram:LineTotalAmount>20.00</ram:LineTotalAmount>", "<ram:LineTotalAmount>25.00</ram:LineTotalAmount>", 2)

	e, err := cii.Parse([]byte(xml))
	require.NoError(t, err)
	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.Len(t, inv.Lines, 6)
	assert.Equal(t, "Service bundle", inv.Lines[3].Item.Name)
	assert.Empty(t, inv.Lines[3].Breakdown)
	assert.Equal(t, "Installation", inv.Lines[4].Item.Name)
}

// TestParseSubInvoiceLinesCycle covers lines naming each other as parents,
// which never lead back to a top-level line: they are still read.
func TestParseSubInvoiceLinesCycle(t *testing.T) {
	data, err := os.ReadFile(dataPath(pathParse, "sub-invoice-lines.xml"))
	require.NoError(t, err)
	xml := strings.Replace(string(data),
		"<ram:LineID>1</ram:LineID><ram:LineStatusReasonCode>GROUP",
		"<ram:LineID>1</ram:LineID><ram:ParentLineID>2</ram:ParentLineID><ram:LineStatusReasonCode>GROUP", 1)

	e, err := cii.Parse([]byte(xml))
	require.NoError(t, err)
	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	assert.Len(t, inv.Lines, 5)
	assert.Equal(t, "170.00", inv.Totals.Sum.String())
}
