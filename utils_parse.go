package cii

import (
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// goblUnit resolves a UN/ECE unit code into the GOBL unit key it stands for,
// keeping the code itself in the extensions: the document stated it, and for
// a code GOBL has no key for it is all there is.
func goblUnit(ext tax.Extensions, code cbc.Code) (cbc.Key, tax.Extensions) {
	return untdid.NormalizeUnit(cbc.KeyEmpty, ext.Set(untdid.ExtKeyUnit, code))
}
