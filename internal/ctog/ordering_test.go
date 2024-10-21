package cii_test

import (
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl.cii/test"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGOrdering(t *testing.T) {
	t.Run("CII_example7.xml", func(t *testing.T) {
		doc, err := test.LoadTestXMLDoc("CII_example7.xml")
		require.NoError(t, err)

		goblEnv, err := cii.NewGOBLFromCII(doc)
		require.NoError(t, err)

		invoice, ok := goblEnv.Extract().(*bill.Invoice)
		require.True(t, ok, "Document should be an invoice")

		require.NotNil(t, invoice.Ordering, "Ordering should not be nil")
		require.NotNil(t, invoice.Ordering.Period, "OrderingPeriod should not be nil")
		assert.Equal(t, "2013-01-01", invoice.Ordering.Period.Start.String(), "OrderingPeriod start date should match")
		assert.Equal(t, "2013-12-31", invoice.Ordering.Period.End.String(), "OrderingPeriod end date should match")
	})
	t.Run("CII_example8.xml", func(t *testing.T) {
		doc, err := test.LoadTestXMLDoc("CII_example8.xml")
		require.NoError(t, err)

		goblEnv, err := cii.NewGOBLFromCII(doc)
		require.NoError(t, err)

		invoice, ok := goblEnv.Extract().(*bill.Invoice)
		require.True(t, ok, "Document should be an invoice")

		require.NotNil(t, invoice.Ordering, "Ordering should not be nil")
		require.NotNil(t, invoice.Ordering.Period, "OrderingPeriod should not be nil")
		assert.Equal(t, "2014-08-01", invoice.Ordering.Period.Start.String(), "OrderingPeriod start date should match")
		assert.Equal(t, "2014-08-31", invoice.Ordering.Period.End.String(), "OrderingPeriod end date should match")
	})

}
