package cii

import (
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// goblUnit resolves a UN/ECE unit code into the GOBL unit key it stands for,
// or, when GOBL cannot express it, keeps the code in the extensions with no
// unit alongside it.
func goblUnit(ext tax.Extensions, code cbc.Code) (cbc.Key, tax.Extensions) {
	return untdid.NormalizeUnit(cbc.KeyEmpty, ext.Set(untdid.ExtKeyUnit, code))
}
