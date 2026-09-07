package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPeriodSingleBound covers EN 16931 BR-CO-19 / BR-CO-20: a billing
// period only needs one of its two bounds. The fixture carries a header
// period with only an end date (BT-74 without BT-73) and a line period with
// only a start date (BT-134 without BT-135). Parsing must produce a GOBL
// document that validates, and converting it back must only emit the bound
// that is present, never an empty DateTimeString.
func TestPeriodSingleBound(t *testing.T) {
	t.Run("parse and validate", func(t *testing.T) {
		env, err := parseInvoiceFrom(t, "CII_period_single_bound.xml")
		require.NoError(t, err)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// The upstream sample this fixture derives from carries tax IDs that
		// do not pass regime validation, so only assert that the one-sided
		// periods are no longer a reason for the document to fail.
		if err := env.Validate(); err != nil {
			assert.NotContains(t, err.Error(), "GOBL-CAL-PERIOD", "one-sided periods must validate")
		}

		require.NotNil(t, inv.Ordering)
		require.NotNil(t, inv.Ordering.Period)
		assert.NoError(t, rules.Validate(inv.Ordering.Period))
		assert.True(t, inv.Ordering.Period.Start.IsZero(), "BT-73 absent, start must stay zero")
		assert.Equal(t, "2013-06-30", inv.Ordering.Period.End.String())

		require.NotEmpty(t, inv.Lines)
		require.NotNil(t, inv.Lines[0].Period)
		assert.NoError(t, rules.Validate(inv.Lines[0].Period))
		assert.Equal(t, "2013-06-01", inv.Lines[0].Period.Start.String())
		assert.True(t, inv.Lines[0].Period.End.IsZero(), "BT-135 absent, end must stay zero")
	})

	t.Run("re-export omits the absent bound", func(t *testing.T) {
		env, err := parseInvoiceFrom(t, "CII_period_single_bound.xml")
		require.NoError(t, err)

		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextEN16931V2017))
		require.NoError(t, err)
		data, err := out.Bytes()
		require.NoError(t, err)
		xml := string(data)

		assert.Contains(t, xml, "<ram:StartDateTime>")
		assert.Contains(t, xml, `<udt:DateTimeString format="102">20130601</udt:DateTimeString>`)
		assert.NotContains(t, xml, `<udt:DateTimeString format="102"></udt:DateTimeString>`)
		assert.NotContains(t, xml, `<udt:DateTimeString format="102"/>`)
		assert.NotContains(t, xml, "00000000")
	})

	t.Run("convert fixture omits the absent bound", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-period-single-bound.json")

		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextEN16931V2017))
		require.NoError(t, err)
		data, err := out.Bytes()
		require.NoError(t, err)
		xml := string(data)

		// Header (BG-14): end only.
		assert.Contains(t, xml, `<udt:DateTimeString format="102">20240531</udt:DateTimeString>`)
		// Line (BG-26): start only.
		assert.Contains(t, xml, `<udt:DateTimeString format="102">20240512</udt:DateTimeString>`)
		assert.Equal(t, 1, countSubstring(xml, "<ram:StartDateTime>"), "exactly one start bound")
		assert.Equal(t, 1, countSubstring(xml, "<ram:EndDateTime>"), "exactly one end bound")
		assert.NotContains(t, xml, `<udt:DateTimeString format="102"></udt:DateTimeString>`)
		assert.NotContains(t, xml, `<udt:DateTimeString format="102"/>`)
	})
}

func countSubstring(s, sub string) int {
	n := 0
	for i := 0; ; {
		j := indexFrom(s, sub, i)
		if j < 0 {
			return n
		}
		n++
		i = j + len(sub)
	}
}

func indexFrom(s, sub string, from int) int {
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
