package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// vatRate is a small helper: a nil percent means the rate is exempt.
func vatRate(base, amount num.Amount, percent *num.Percentage) *tax.RateTotal {
	return &tax.RateTotal{Base: base, Percent: percent, Amount: amount}
}

func amt(v int64) num.Amount { return num.MakeAmount(v, 2) }

func pct(v int64) *num.Percentage {
	p := num.MakePercentage(v, 2)
	return &p
}

// Expected CDV values, named to keep the case table readable (and to
// satisfy goconst, which flags repeated string literals).
const (
	rate20   = "20.00"  // standard rate
	rateZero = "0.00"   // zero rate / exempt collapse
	grossStd = "600.00" // 500.00 base + 100.00 VAT @ 20%
	grossRed = "220.00" // 200.00 base + 20.00 VAT @ 10%
	grossZro = "300.00" // 300.00 base, no VAT
)

// TestCDARPaymentVATSplitRoundTrip exercises the cashed-amount ⇄ VAT-rate
// mapping for an encaissement (212). The CDV expresses the cashed amount as
// one MEN characteristic per rate — each carrying that rate's gross (TTC) and
// its percentage — with no separate global total. On the GOBL side the
// payment line holds a single Amount (the total cashed) plus N rates; Amount
// is therefore the sum of the per-rate grosses. Exempt and zero rates both
// collapse to 0.00 in the CDV.
func TestCDARPaymentVATSplitRoundTrip(t *testing.T) {
	type wantChar struct {
		amount  string
		percent string
	}
	cases := []struct {
		name       string
		rates      []*tax.RateTotal
		wantChars  []wantChar
		wantAmount string // expected parsed line.Amount = Σ grosses
	}{
		{
			name:       "single rate",
			rates:      []*tax.RateTotal{vatRate(amt(50000), amt(10000), pct(20))}, // 500 + 100
			wantChars:  []wantChar{{grossStd, rate20}},
			wantAmount: grossStd,
		},
		{
			name: "multiple rates",
			rates: []*tax.RateTotal{
				vatRate(amt(50000), amt(10000), pct(20)), // gross 600.00
				vatRate(amt(20000), amt(2000), pct(10)),  // gross 220.00
			},
			wantChars:  []wantChar{{grossStd, rate20}, {grossRed, "10.00"}},
			wantAmount: "820.00",
		},
		{
			name:       "zero rate",
			rates:      []*tax.RateTotal{vatRate(amt(30000), amt(0), pct(0))}, // gross 300.00
			wantChars:  []wantChar{{grossZro, rateZero}},
			wantAmount: grossZro,
		},
		{
			name:       "exempt (nil percent)",
			rates:      []*tax.RateTotal{vatRate(amt(30000), amt(0), nil)}, // gross 300.00
			wantChars:  []wantChar{{grossZro, rateZero}},
			wantAmount: grossZro,
		},
		{
			name: "mixed taxed and exempt",
			rates: []*tax.RateTotal{
				vatRate(amt(50000), amt(10000), pct(20)), // gross 600.00
				vatRate(amt(30000), amt(0), nil),         // exempt, gross 300.00
			},
			wantChars:  []wantChar{{grossStd, rate20}, {grossZro, rateZero}},
			wantAmount: "900.00",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pmt := buildSyntheticPayment(t, num.Amount{})
			pmt.Lines[0].Tax = &tax.Total{
				Categories: []*tax.CategoryTotal{{Code: tax.CategoryVAT, Rates: tc.rates}},
			}

			// --- generate: one MEN characteristic per rate ---
			cdar, err := cii.NewCDARFromPayment(pmt, cii.ContextCDARFlow6)
			require.NoError(t, err)
			ref := cdar.AcknowledgementDocuments[0].ReferenceReferencedDocument[0]
			require.Len(t, ref.SpecifiedDocumentStatuses, 1)
			chars := ref.SpecifiedDocumentStatuses[0].SpecifiedDocumentCharacteristics
			require.Len(t, chars, len(tc.wantChars), "one characteristic per rate")
			for i, wc := range tc.wantChars {
				assert.Equal(t, "MEN", chars[i].TypeCode)
				require.NotNil(t, chars[i].ValueAmount)
				assert.Equal(t, wc.amount, chars[i].ValueAmount.Value, "MDT-215 gross per rate")
				assert.Equal(t, wc.percent, chars[i].ValuePercent, "MDT-224 rate (exempt→0.00)")
			}

			// --- parse: Amount is the sum of the grosses, rates recovered ---
			data, err := cdar.Bytes()
			require.NoError(t, err)
			out, err := cii.ParseCDARPayment(data)
			require.NoError(t, err)
			require.Len(t, out.Lines, 1)
			assert.Equal(t, tc.wantAmount, out.Lines[0].Amount.String(),
				"line.Amount must be the total cashed (Σ per-rate grosses)")

			require.NotNil(t, out.Lines[0].Tax)
			require.Len(t, out.Lines[0].Tax.Categories, 1)
			gotRates := out.Lines[0].Tax.Categories[0].Rates
			require.Len(t, gotRates, len(tc.wantChars), "one rate recovered per characteristic")
			for i, wc := range tc.wantChars {
				require.NotNil(t, gotRates[i].Percent, "parse reconstructs a concrete rate (0.00 for exempt)")
				assert.Equal(t, wc.percent, gotRates[i].Percent.Amount().Rescale(2).String())
			}
		})
	}
}

// TestCDARPaymentAmountDerivedFromRates proves the CDV amount comes from the
// tax breakdown, not from line.Amount: a deliberately wrong line.Amount does
// not leak into the characteristics.
func TestCDARPaymentAmountDerivedFromRates(t *testing.T) {
	pmt := buildSyntheticPayment(t, num.Amount{})
	pmt.Lines[0].Amount = amt(999999) // bogus; must be ignored on generate
	pmt.Lines[0].Tax = &tax.Total{
		Categories: []*tax.CategoryTotal{{Code: tax.CategoryVAT, Rates: []*tax.RateTotal{
			vatRate(amt(50000), amt(10000), pct(20)), // gross 600.00
			vatRate(amt(20000), amt(2000), pct(10)),  // gross 220.00
		}}},
	}

	cdar, err := cii.NewCDARFromPayment(pmt, cii.ContextCDARFlow6)
	require.NoError(t, err)
	chars := cdar.AcknowledgementDocuments[0].ReferenceReferencedDocument[0].
		SpecifiedDocumentStatuses[0].SpecifiedDocumentCharacteristics
	require.Len(t, chars, 2)
	assert.Equal(t, grossStd, chars[0].ValueAmount.Value)
	assert.Equal(t, grossRed, chars[1].ValueAmount.Value)

	data, err := cdar.Bytes()
	require.NoError(t, err)
	out, err := cii.ParseCDARPayment(data)
	require.NoError(t, err)
	assert.Equal(t, "820.00", out.Lines[0].Amount.String(),
		"parsed Amount is the Σ of grosses, not the bogus input")
}
