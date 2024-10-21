package cii_test

import (
	"testing"

	gtoc "github.com/invopop/gobl.cii/internal/gtoc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func TestFindTaxCode(t *testing.T) {
	t.Run("should return correct tax category", func(t *testing.T) {
		taxCode := gtoc.FindTaxCode(tax.RateStandard)

		assert.Equal(t, gtoc.StandardSalesTax, taxCode)
	})

	t.Run("should return zero tax category", func(t *testing.T) {
		taxCode := gtoc.FindTaxCode(tax.RateZero)

		assert.Equal(t, gtoc.ZeroRatedGoodsTax, taxCode)
	})

	t.Run("should return zero tax category", func(t *testing.T) {
		taxCode := gtoc.FindTaxCode(tax.RateExempt)

		assert.Equal(t, gtoc.TaxExempt, taxCode)
	})
}
