package cii_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGOrdering(t *testing.T) {
	t.Run("CII_example7.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_example7.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Ordering, "Ordering should not be nil")
		require.NotNil(t, inv.Ordering.Period, "OrderingPeriod should not be nil")
		assert.Equal(t, "2013-01-01", inv.Ordering.Period.Start.String(), "OrderingPeriod start date should match")
		assert.Equal(t, "2013-12-31", inv.Ordering.Period.End.String(), "OrderingPeriod end date should match")
	})
	t.Run("CII_example8.xml", func(t *testing.T) {
		e, err := parseInvoiceFrom(t, "CII_example8.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Ordering, "Ordering should not be nil")
		require.NotNil(t, inv.Ordering.Period, "OrderingPeriod should not be nil")
		assert.Equal(t, "2014-08-01", inv.Ordering.Period.Start.String(), "OrderingPeriod start date should match")
		assert.Equal(t, "2014-08-31", inv.Ordering.Period.End.String(), "OrderingPeriod end date should match")
	})

}
