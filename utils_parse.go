package cii

import (
	"regexp"
	"strings"

	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

// goblItemUnit records a UN/ECE unit code on the item. The raw code is kept
// in the untdid-unit extension and mapped to a GOBL unit key when one exists.
func goblItemUnit(item *org.Item, code cbc.Code) {
	if code == cbc.CodeEmpty {
		return
	}
	item.Ext = item.Ext.Set(untdid.ExtKeyUnit, code)
	if unit := en16931.UnitFromUNTDID(code); unit != cbc.KeyEmpty {
		item.Unit = unit
	}
}

func formatKey(key string) cbc.Key {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-+]`)
	key = re.ReplaceAllString(key, "")
	key = strings.Trim(key, "-+")
	re = regexp.MustCompile(`[-+]{2,}`)
	key = re.ReplaceAllString(key, "-")
	return cbc.Key(key)
}
