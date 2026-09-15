package cii_test

import (
	"strings"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	emptyDateTime      = `<udt:DateTimeString format="102"></udt:DateTimeString>`
	emptyDateTimeShort = `<udt:DateTimeString format="102"/>`
)

// A period with a single bound (BR-CO-19 / BR-CO-20) must parse, validate
// and export without an empty DateTimeString for the missing side.
func TestPeriodSingleBound(t *testing.T) {
	t.Run("parse", func(t *testing.T) {
		env, err := parseInvoiceFrom(t, "CII_period_single_bound.xml")
		require.NoError(t, err)
		inv := env.Extract().(*bill.Invoice)

		require.NotNil(t, inv.Ordering.Period)
		assert.NoError(t, rules.Validate(inv.Ordering.Period))
		assert.Nil(t, inv.Ordering.Period.Start)
		assert.Equal(t, "2013-06-30", inv.Ordering.Period.End.String())

		require.NotNil(t, inv.Lines[0].Period)
		assert.NoError(t, rules.Validate(inv.Lines[0].Period))
		assert.Equal(t, "2013-06-01", inv.Lines[0].Period.Start.String())
		assert.Nil(t, inv.Lines[0].Period.End)
	})

	t.Run("round trip", func(t *testing.T) {
		env, err := parseInvoiceFrom(t, "CII_period_single_bound.xml")
		require.NoError(t, err)

		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextEN16931V2017))
		require.NoError(t, err)
		data, err := out.Bytes()
		require.NoError(t, err)
		xml := string(data)

		assert.Equal(t, 1, strings.Count(xml, "<ram:StartDateTime>"))
		assert.Equal(t, 1, strings.Count(xml, "<ram:EndDateTime>"))
		assert.Contains(t, xml, `format="102">20130601<`)
		assert.Contains(t, xml, `format="102">20130630<`)
		assert.NotContains(t, xml, emptyDateTime)
		assert.NotContains(t, xml, emptyDateTimeShort)
	})

	t.Run("convert", func(t *testing.T) {
		env := loadEnvelope(t, "en16931/invoice-period-single-bound.json")

		out, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextEN16931V2017))
		require.NoError(t, err)
		data, err := out.Bytes()
		require.NoError(t, err)
		xml := string(data)

		assert.Equal(t, 1, strings.Count(xml, "<ram:StartDateTime>"))
		assert.Equal(t, 1, strings.Count(xml, "<ram:EndDateTime>"))
		assert.NotContains(t, xml, emptyDateTime)
		assert.NotContains(t, xml, emptyDateTimeShort)
	})
}
