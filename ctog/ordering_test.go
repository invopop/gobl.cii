package ctog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGOrdering(t *testing.T) {
	t.Run("CII_example7.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("CII_example7.xml")
		require.NoError(t, err)

		c := NewConverter()
		err = c.NewInvoice(doc)
		require.NoError(t, err)

		inv := c.GetInvoice()

		require.NotNil(t, inv.Ordering, "Ordering should not be nil")
		require.NotNil(t, inv.Ordering.Period, "OrderingPeriod should not be nil")
		assert.Equal(t, "2013-01-01", inv.Ordering.Period.Start.String(), "OrderingPeriod start date should match")
		assert.Equal(t, "2013-12-31", inv.Ordering.Period.End.String(), "OrderingPeriod end date should match")
	})
	t.Run("CII_example8.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("CII_example8.xml")
		require.NoError(t, err)

		c := NewConverter()
		err = c.NewInvoice(doc)
		require.NoError(t, err)

		inv := c.GetInvoice()

		require.NotNil(t, inv.Ordering, "Ordering should not be nil")
		require.NotNil(t, inv.Ordering.Period, "OrderingPeriod should not be nil")
		assert.Equal(t, "2014-08-01", inv.Ordering.Period.Start.String(), "OrderingPeriod start date should match")
		assert.Equal(t, "2014-08-31", inv.Ordering.Period.End.String(), "OrderingPeriod end date should match")
	})

}
