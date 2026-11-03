package cii

import (
	"testing"

	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDeclared = "20.00"
	testBasis    = "200.00"
	testTenth    = "10.00"
)

// GOBL requires a percentage wherever a basis is set, so dropping a percentage
// that disagrees with the declared amount has to drop the basis with it.
// Otherwise the document converts happily and then fails validation at signing
// with GOBL-BILL-LINEDISCOUNT-01.
func TestDeclaredAmountWinsWithoutLeavingABasis(t *testing.T) {
	t.Run("line discount", func(t *testing.T) {
		d, err := goblNewLineDiscount(&AllowanceCharge{
			Amount:  testDeclared,
			Base:    testBasis,
			Percent: testAmountHalf, // 50% of 200.00 is 100.00, not 20.00
		})
		require.NoError(t, err)
		assert.Equal(t, num.MakeAmount(2000, 2), d.Amount)
		assert.Nil(t, d.Percent)
		assert.Nil(t, d.Base)
	})

	t.Run("line charge", func(t *testing.T) {
		c, err := goblNewLineCharge(&AllowanceCharge{
			Amount:  testDeclared,
			Base:    testBasis,
			Percent: testAmountHalf,
		})
		require.NoError(t, err)
		assert.Nil(t, c.Percent)
		assert.Nil(t, c.Base)
	})

	t.Run("document discount", func(t *testing.T) {
		d, err := goblNewDiscount(&AllowanceCharge{
			Amount:  testDeclared,
			Base:    testBasis,
			Percent: testAmountHalf,
		}, nil)
		require.NoError(t, err)
		assert.Nil(t, d.Percent)
		assert.Nil(t, d.Base)
	})

	t.Run("a basis with no percentage at all is dropped", func(t *testing.T) {
		// Nothing to apply it to, and GOBL would reject it.
		d, err := goblNewLineDiscount(&AllowanceCharge{
			Amount: testDeclared,
			Base:   testBasis,
		})
		require.NoError(t, err)
		assert.Nil(t, d.Base)
		assert.Nil(t, d.Percent)
	})

	t.Run("a basis that reproduces the amount is kept", func(t *testing.T) {
		d, err := goblNewLineDiscount(&AllowanceCharge{
			Amount:  testDeclared,
			Base:    testBasis,
			Percent: testTenth, // 10% of 200.00 is 20.00
		})
		require.NoError(t, err)
		require.NotNil(t, d.Percent)
		require.NotNil(t, d.Base)
		assert.Equal(t, num.MakeAmount(20000, 2), *d.Base)
	})
}
