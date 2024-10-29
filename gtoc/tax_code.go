package gtoc

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

const (
	StandardSalesTax  = "S"
	ZeroRatedGoodsTax = "Z"
	TaxExempt         = "E"
)

func FindTaxCode(taxRate cbc.Key) string {
	switch taxRate {
	case tax.RateStandard:
		return StandardSalesTax
	case tax.RateZero:
		return ZeroRatedGoodsTax
	case tax.RateExempt:
		return TaxExempt
	}

	return StandardSalesTax
}
