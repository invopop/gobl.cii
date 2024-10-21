package cii

import (
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

const (
	keyPaymentMeansSEPACreditTransfer cbc.Key = "sepa-credit-transfer"
	keyPaymentMeansSEPADirectDebit    cbc.Key = "sepa-direct-debit"
)

const (
	PaymentMeansCash               = "10"
	PaymentMeansCheque             = "20"
	PaymentMeansCreditTransfer     = "30"
	PaymentMeansBankAccount        = "42"
	PaymentMeansCard               = "48"
	PaymentMeansDirectDebit        = "49"
	PaymentMeansStandingOrder      = "57"
	PaymentMeansSEPACreditTransfer = "58"
	PaymentMeansSEPADirectDebit    = "59"
	PaymentMeansReport             = "97"
)

const (
	StandardSalesTax  = "S"
	ZeroRatedGoodsTax = "Z"
	TaxExempt         = "E"
)

const (
	keyInvoiceTypeSelfBilled               cbc.Key = "self-billed"
	keyInvoiceTypePartial                  cbc.Key = "partial"
	keyInvoiceTypePartialConstruction      cbc.Key = "partial-construction"
	keyInvoiceTypePartialFinalConstruction cbc.Key = "partial-final-construction"
	keyInvoiceTypeFinalConstruction        cbc.Key = "final-construction"
)

const (
	InvoiceTypeProforma                 = "325"
	InvoiceTypeStandard                 = "380"
	InvoiceTypeCreditNote               = "381"
	InvoiceTypeDebitNote                = "383"
	InvoiceTypeCorrective               = "384"
	InvoiceTypeSelfBilled               = "389"
	InvoiceTypePartial                  = "326"
	InvoiceTypePartialConstruction      = "875"
	InvoiceTypePartialFinalConstruction = "876"
	InvoiceTypeFinalConstruction        = "877"
)

// Convert a date string to a cal.Date
func ParseDate(date string) cal.Date {
	t, err := time.Parse("20060102", date)
	if err != nil {
		return cal.Date{}
	}

	return cal.MakeDate(t.Year(), t.Month(), t.Day())
}

// Map X-Rechnung rate to GOBL equivalent
func FindTaxKey(taxType string) cbc.Key {
	switch taxType {
	case StandardSalesTax:
		return tax.RateStandard
	case ZeroRatedGoodsTax:
		return tax.RateZero
	case TaxExempt:
		return tax.RateExempt
	}
	return tax.RateStandard
}

// Map CII invoice type to GOBL equivalent
// Source https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred1001.htm
func TypeCodeParse(typeCode string) cbc.Key {
	switch typeCode {
	case InvoiceTypeStandard:
		return bill.InvoiceTypeStandard
	case InvoiceTypeCreditNote:
		return bill.InvoiceTypeCreditNote
	case InvoiceTypeCorrective:
		return bill.InvoiceTypeCorrective
	case InvoiceTypeProforma:
		return bill.InvoiceTypeProforma
	case InvoiceTypeDebitNote:
		return bill.InvoiceTypeDebitNote
	case InvoiceTypeSelfBilled:
		return keyInvoiceTypeSelfBilled
	case InvoiceTypePartial:
		return keyInvoiceTypePartial
	case InvoiceTypePartialConstruction:
		return keyInvoiceTypePartialConstruction
	case InvoiceTypePartialFinalConstruction:
		return keyInvoiceTypePartialFinalConstruction
	case InvoiceTypeFinalConstruction:
		return keyInvoiceTypeFinalConstruction
	}
	return bill.InvoiceTypeOther
}

// Map UN/ECE code to GOBL equivalent
func UnitFromUNECE(unece cbc.Code) org.Unit {
	for _, def := range org.UnitDefinitions {
		if def.UNECE == unece {
			return def.Unit
		}
	}
	// If no match is found, return the original UN/ECE code as a Unit
	return org.Unit(unece)
}

// Map CII payment means to GOBL equivalent
// Source https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred4461.htm
func PaymentMeansTypeCodeParse(typeCode string) cbc.Key {
	switch typeCode {
	case PaymentMeansCash:
		return pay.MeansKeyCash
	case PaymentMeansCheque:
		return pay.MeansKeyCheque
	case PaymentMeansCreditTransfer:
		return pay.MeansKeyCreditTransfer
	case PaymentMeansBankAccount:
		return pay.MeansKeyDebitTransfer
	case PaymentMeansCard:
		return pay.MeansKeyCard
	case PaymentMeansSEPACreditTransfer:
		return keyPaymentMeansSEPACreditTransfer
	case PaymentMeansSEPADirectDebit:
		return keyPaymentMeansSEPADirectDebit
	case PaymentMeansDirectDebit:
		return pay.MeansKeyDirectDebit
	default:
		return pay.MeansKeyOther
	}
}
