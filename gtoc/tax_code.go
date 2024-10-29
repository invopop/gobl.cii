package gtoc

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

const (
	standardSalesTax  = "S"
	zeroRatedGoodsTax = "Z"
	taxExempt         = "E"
)

func findTaxCode(taxRate cbc.Key) string {
	switch taxRate {
	case tax.RateStandard:
		return standardSalesTax
	case tax.RateZero:
		return zeroRatedGoodsTax
	case tax.RateExempt:
		return taxExempt
	}

	return standardSalesTax
}
