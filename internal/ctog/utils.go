package ctog

import (
	"regexp"
	"strings"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

// Currently supported payment means and invoice type keys
var (
	paymentMeansMap = map[string]cbc.Key{
		"10": pay.MeansKeyCash,
		"20": pay.MeansKeyCheque,
		"30": pay.MeansKeyCreditTransfer,
		"42": pay.MeansKeyDebitTransfer,
		"48": pay.MeansKeyCard,
		"49": pay.MeansKeyDirectDebit,
		"58": pay.MeansKeyCreditTransfer.With(pay.MeansKeySEPA),
		"59": pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA),
	}

	invoiceTypeMap = map[string]cbc.Key{
		"325": bill.InvoiceTypeProforma,
		"380": bill.InvoiceTypeStandard,
		"381": bill.InvoiceTypeCreditNote,
		"383": bill.InvoiceTypeDebitNote,
		"384": bill.InvoiceTypeCorrective,
		"389": bill.InvoiceTypeStandard.With(tax.TagSelfBilled),
		"326": bill.InvoiceTypeStandard.With(tax.TagPartial),
	}
)

// ParseDate converts a date string to a cal.Date
func ParseDate(date string) (cal.Date, error) {
	t, err := time.Parse("20060102", date)
	if err != nil {
		return cal.Date{}, err
	}

	return cal.MakeDate(t.Year(), t.Month(), t.Day()), nil
}

// TypeCodeParse maps a CII invoice type to a GOBL equivalent
// Source https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred1001.htm
func TypeCodeParse(typeCode string) cbc.Key {
	if val, ok := invoiceTypeMap[typeCode]; ok {
		return val
	}
	return bill.InvoiceTypeOther
}

// unitFromUNECE maps a UN/ECE code to a GOBL unit
func unitFromUNECE(unece *cbc.Code) *org.Unit {
	if unece == nil {
		return nil
	}
	for _, def := range org.UnitDefinitions {
		if def.UNECE == *unece {
			return &def.Unit
		}
	}
	// If no match is found, return the original UN/ECE code as a Unit
	unit := org.Unit(*unece)
	return &unit
}

// paymentMeansCode maps a CII payment means to a GOBL equivalent
// Source https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred4461.htm
func paymentMeansCode(code string) cbc.Key {
	if val, ok := paymentMeansMap[code]; ok {
		return val
	}
	return pay.MeansKeyAny
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
