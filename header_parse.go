package cii

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

var invoiceTypeMap = map[string]cbc.Key{
	"325": bill.InvoiceTypeProforma,
	"380": bill.InvoiceTypeStandard,
	"381": bill.InvoiceTypeCreditNote,
	"383": bill.InvoiceTypeDebitNote,
	"384": bill.InvoiceTypeCorrective,
	"389": bill.InvoiceTypeStandard,
	"326": bill.InvoiceTypeStandard,
}

// invoiceTagMap holds the tags a type code adds to its GOBL type
var invoiceTagMap = map[string][]cbc.Key{
	"389": {tax.TagSelfBilled},
	"326": {tax.TagPartial},
}

// typeCodeParse maps a CII invoice type to a GOBL type and the tags it implies
// Source https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred1001.htm
func typeCodeParse(typeCode string) (cbc.Key, []cbc.Key) {
	if val, ok := invoiceTypeMap[typeCode]; ok {
		return val, invoiceTagMap[typeCode]
	}
	return bill.InvoiceTypeOther, nil
}
