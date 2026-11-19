package cii

import (
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
)

// rescaleToCurrency rounds the amount to the natural precision of the given
// currency code (2 for EUR, 0 for JPY). Falls back to the amount's existing
// precision if the currency code is unknown.
func rescaleToCurrency(a num.Amount, ccy string) string {
	if def := currency.Code(ccy).Def(); def != nil {
		return def.Rescale(a).String()
	}
	return a.String()
}
