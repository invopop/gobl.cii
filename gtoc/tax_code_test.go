package gtoc

import (
	"testing"

	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func TestFindTaxCode(t *testing.T) {
	t.Run("should return correct tax category", func(t *testing.T) {
		taxCode := FindTaxCode(tax.RateStandard)

		assert.Equal(t, StandardSalesTax, taxCode)
	})

	t.Run("should return zero tax category", func(t *testing.T) {
		taxCode := FindTaxCode(tax.RateZero)

		assert.Equal(t, ZeroRatedGoodsTax, taxCode)
	})

	t.Run("should return zero tax category", func(t *testing.T) {
		taxCode := FindTaxCode(tax.RateExempt)

		assert.Equal(t, TaxExempt, taxCode)
	})
}
