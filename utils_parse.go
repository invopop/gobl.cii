package cii

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

// unitFromUNECE maps a UN/ECE code to a GOBL unit
func goblUnitFromUNECE(unece cbc.Code) org.Unit {
	if unece == cbc.CodeEmpty {
		return org.UnitEmpty
	}
	for _, def := range org.UnitDefinitions {
		if def.UNECE == unece {
			return def.Unit
		}
	}
	// If no match is found, return the original UN/ECE code as a Unit
	unit := org.Unit(unece)
	return unit
}
