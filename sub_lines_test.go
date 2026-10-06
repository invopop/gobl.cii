package cii_test

import (
	"os"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	lineStatusGroup       = "GROUP"
	lineStatusDetail      = "DETAIL"
	lineStatusInformation = "INFORMATION"
)

const fixtureFacturXDE = "facturx/invoice-de-de.json"

// breakdownEnvelope loads a fixture and gives its first line a breakdown: two
// priced sub-lines, one of them discounted, and one without a price.
func breakdownEnvelope(t *testing.T, fixture string) *gobl.Envelope {
	t.Helper()
	env := loadEnvelope(t, fixture)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	design := num.MakeAmount(3000, 2)
	build := num.MakeAmount(6000, 2)
	inv.Lines[0].Discounts, inv.Lines[0].Charges = nil, nil
	inv.Lines[0].Breakdown = []*bill.SubLine{
		{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Design", Price: &design}},
		{
			Quantity:  num.MakeAmount(2, 0),
			Item:      &org.Item{Name: "Build", Price: &build},
			Discounts: []*bill.LineDiscount{{Amount: num.MakeAmount(500, 2), Reason: "Promotion"}},
		},
		{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Sample, no charge"}},
	}
	require.NoError(t, env.Calculate())
	return env
}

func TestSubLinesConvert(t *testing.T) {
	t.Run("extended profile writes sub-invoice lines", func(t *testing.T) {
		doc, err := cii.ConvertInvoice(breakdownEnvelope(t, fixtureFacturXDE), cii.WithContext(cii.ContextFacturXExtendedV1))
		require.NoError(t, err)
		lines := doc.Transaction.Lines
		require.Len(t, lines, 4)

		// The parent quantity is 20, and each sub-line counts per unit of it.
		group := lines[0]
		assert.Equal(t, "1", group.LineDoc.ID)
		assert.Equal(t, lineStatusGroup, group.LineDoc.LineStatusReasonCode)
		assert.Empty(t, group.LineDoc.ParentLineID)
		assert.Empty(t, group.TradeSettlement.ApplicableTradeTax)
		assert.Equal(t, "145.00", group.Agreement.NetPrice.Amount)
		assert.Equal(t, "20", group.Quantity.Quantity.Amount)
		assert.Equal(t, "2900.00", group.TradeSettlement.Sum.Amount)

		design := lines[1]
		assert.Equal(t, "1.1", design.LineDoc.ID)
		assert.Equal(t, "1", design.LineDoc.ParentLineID)
		assert.Equal(t, lineStatusDetail, design.LineDoc.LineStatusReasonCode)
		assert.Equal(t, "Design", design.Product.Name)
		assert.Equal(t, "20", design.Quantity.Quantity.Amount)
		assert.Equal(t, "30.00", design.Agreement.NetPrice.Amount)
		assert.Equal(t, "600.00", design.TradeSettlement.Sum.Amount)
		require.Len(t, design.TradeSettlement.ApplicableTradeTax, 1)
		assert.Equal(t, "S", design.TradeSettlement.ApplicableTradeTax[0].CategoryCode)

		build := lines[2]
		assert.Equal(t, "1.2", build.LineDoc.ID)
		assert.Equal(t, "40", build.Quantity.Quantity.Amount)
		require.Len(t, build.TradeSettlement.AllowanceCharge, 1)
		assert.Equal(t, "100.00", build.TradeSettlement.AllowanceCharge[0].Amount)
		assert.Equal(t, "2300.00", build.TradeSettlement.Sum.Amount)

		info := lines[3]
		assert.Equal(t, "1.3", info.LineDoc.ID)
		assert.Equal(t, lineStatusInformation, info.LineDoc.LineStatusReasonCode)
		assert.Equal(t, rateZero, info.Agreement.NetPrice.Amount)
		assert.Equal(t, rateZero, info.TradeSettlement.Sum.Amount)
	})

	t.Run("other profiles write the line alone", func(t *testing.T) {
		doc, err := cii.ConvertInvoice(breakdownEnvelope(t, "en16931/invoice-de-de.json"))
		require.NoError(t, err)
		require.Len(t, doc.Transaction.Lines, 1)
		assert.Empty(t, doc.Transaction.Lines[0].LineDoc.LineStatusReasonCode)
		assert.Equal(t, "2900.00", doc.Transaction.Lines[0].TradeSettlement.Sum.Amount)
	})

	t.Run("a line with its own discount is written alone", func(t *testing.T) {
		env := breakdownEnvelope(t, fixtureFacturXDE)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Lines[0].Discounts = []*bill.LineDiscount{{Amount: num.MakeAmount(1000, 2), Reason: "Loyalty"}}
		require.NoError(t, env.Calculate())

		doc, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextFacturXExtendedV1))
		require.NoError(t, err)
		require.Len(t, doc.Transaction.Lines, 1)
	})

	t.Run("unpriced sub-lines describe a line that keeps its price", func(t *testing.T) {
		env := loadEnvelope(t, fixtureFacturXDE)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Lines[0].Breakdown = []*bill.SubLine{
			{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Helmet"}},
		}
		require.NoError(t, env.Calculate())

		doc, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextFacturXExtendedV1))
		require.NoError(t, err)
		lines := doc.Transaction.Lines
		require.Len(t, lines, 2)
		assert.Equal(t, lineStatusDetail, lines[0].LineDoc.LineStatusReasonCode)
		assert.NotEmpty(t, lines[0].TradeSettlement.ApplicableTradeTax)
		assert.Equal(t, "1800.00", lines[0].TradeSettlement.Sum.Amount)
		assert.Equal(t, lineStatusInformation, lines[1].LineDoc.LineStatusReasonCode)
	})
}

// TestSubLinesRoundTrip writes a breakdown out as sub-invoice lines and reads
// it back: the breakdown and every total must survive.
func TestSubLinesRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		env     func(t *testing.T) *gobl.Envelope
		context cii.Context
	}{
		{"invoice-hierarchy", func(t *testing.T) *gobl.Envelope {
			return loadEnvelope(t, "peppol-france-facturx/invoice-hierarchy.json")
		}, cii.ContextPeppolFranceFacturXV1},
		{"invoice-sub-lines", func(t *testing.T) *gobl.Envelope {
			return loadEnvelope(t, "peppol-france-extended/invoice-sub-lines.json")
		}, cii.ContextPeppolFranceExtendedV1},
		{"facturx-extended", func(t *testing.T) *gobl.Envelope {
			return breakdownEnvelope(t, fixtureFacturXDE)
		}, cii.ContextFacturXExtendedV1},
		{"zugferd-extended", func(t *testing.T) *gobl.Envelope {
			return breakdownEnvelope(t, "zugferd/standard-invoice.json")
		}, cii.ContextZUGFeRDExtendedV2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.env(t)
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)

			doc, err := cii.ConvertInvoice(env, cii.WithContext(tt.context))
			require.NoError(t, err)
			data, err := doc.Bytes()
			require.NoError(t, err)

			parsed, err := cii.Parse(data)
			require.NoError(t, err)
			out, ok := parsed.Extract().(*bill.Invoice)
			require.True(t, ok)

			assert.False(t, out.HasTags(tax.TagBypass))
			require.Len(t, out.Lines, len(inv.Lines))
			for i, l := range inv.Lines {
				got := out.Lines[i]
				assert.Equal(t, l.Quantity.String(), got.Quantity.String(), "line %d quantity", i+1)
				assert.Equal(t, l.Total.String(), got.Total.String(), "line %d total", i+1)
				require.Len(t, got.Breakdown, len(l.Breakdown), "line %d breakdown", i+1)
				for j, sl := range l.Breakdown {
					gs := got.Breakdown[j]
					assert.Equal(t, sl.Item.Name, gs.Item.Name)
					assert.True(t, sl.Quantity.Equals(gs.Quantity), "sub-line %d.%d quantity", i+1, j+1)
					if sl.Item.Price == nil {
						assert.Nil(t, gs.Item.Price)
						assert.Nil(t, gs.Total)
						continue
					}
					require.NotNil(t, gs.Item.Price)
					assert.True(t, sl.Item.Price.Equals(*gs.Item.Price), "sub-line %d.%d price", i+1, j+1)
					assert.Equal(t, sl.Total.String(), gs.Total.String(), "sub-line %d.%d total", i+1, j+1)
					assert.Len(t, gs.Discounts, len(sl.Discounts))
				}
			}
			assert.Equal(t, inv.Totals.Sum.String(), out.Totals.Sum.String())
			assert.Equal(t, inv.Totals.Tax.String(), out.Totals.Tax.String())
			assert.Equal(t, inv.Totals.Payable.String(), out.Totals.Payable.String())
		})
	}
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
