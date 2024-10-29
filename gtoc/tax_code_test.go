package gtoc

import (
	"testing"

	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func TestFindTaxCode(t *testing.T) {
	t.Run("should return correct tax category", func(t *testing.T) {
		taxCode := findTaxCode(tax.RateStandard)

		assert.Equal(t, standardSalesTax, taxCode)
	})

	t.Run("should return zero tax category", func(t *testing.T) {
		taxCode := findTaxCode(tax.RateZero)

		assert.Equal(t, zeroRatedGoodsTax, taxCode)
	})

	t.Run("should return zero tax category", func(t *testing.T) {
		taxCode := findTaxCode(tax.RateExempt)

		assert.Equal(t, taxExempt, taxCode)
	})
}
