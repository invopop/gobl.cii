package cii

import (
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// goblUnit preserves the UN/ECE unit code in the given extensions and returns
// them alongside the matching GOBL unit key, which is empty when the code has
// no GOBL equivalent.
func goblUnit(ext tax.Extensions, code cbc.Code) (tax.Extensions, cbc.Key) {
	return ext.Set(untdid.ExtKeyUnit, code), untdid.UnitKey(code)
}
