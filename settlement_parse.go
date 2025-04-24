package cii

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

var paymentMeansMap = map[string]cbc.Key{
	"10": pay.MeansKeyCash,
	"20": pay.MeansKeyCheque,
	"30": pay.MeansKeyCreditTransfer,
	"42": pay.MeansKeyDebitTransfer,
	"48": pay.MeansKeyCard,
	"49": pay.MeansKeyDirectDebit,
	"58": pay.MeansKeyCreditTransfer.With(pay.MeansKeySEPA),
	"59": pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA),
}

func goblNewPaymentDetails(stlm *Settlement) (*bill.PaymentDetails, error) {
	pymt := &bill.PaymentDetails{}

	if stlm.Payee != nil {
		payee := &org.Party{Name: stlm.Payee.Name}
		if stlm.Payee.PostalTradeAddress != nil {
			payee.Addresses = []*org.Address{
				goblNewAddress(stlm.Payee.PostalTradeAddress),
			}
		}
		pymt.Payee = payee
	}
	if stlm.PaymentTerms != nil {
		terms, err := goblNewTerms(stlm)
		if err != nil {
			return nil, err
		}
		pymt.Terms = terms
	}

	if len(stlm.PaymentMeans) > 0 && stlm.PaymentMeans[0].TypeCode != "1" {
		pymt.Instructions = goblNewInstructions(stlm)
	}

	if len(stlm.Advance) > 0 {
		for _, ap := range stlm.Advance {
			amt, err := num.AmountFromString(ap.Amount)
			if err != nil {
				return nil, err
			}
			a := &pay.Advance{
				Amount: amt,
			}
			if ap.Date != nil && ap.Date.DateFormat != nil {
				advancePaymentReceivedDateTime, err := parseDate(ap.Date.DateFormat.Value)
				if err != nil {
					return nil, err
				}
				a.Date = &advancePaymentReceivedDateTime
			}
			pymt.Advances = append(pymt.Advances, a)
		}
	} else if stlm.Summary.TotalPrepaidAmount != "" {
		// Fake an advanced payment so the totals will be re-calculated correctly
		amt, err := num.AmountFromString(stlm.Summary.TotalPrepaidAmount)
		if err != nil {
			return nil, err
		}
		a := &pay.Advance{
			Amount: amt,
		}
		pymt.Advances = append(pymt.Advances, a)
	}

	if pymt.Payee == nil &&
		pymt.Terms == nil &&
		pymt.Instructions == nil &&
		len(pymt.Advances) == 0 {
		return nil, nil
	}

	return pymt, nil
}

func goblNewTerms(settlement *Settlement) (*pay.Terms, error) {
	terms := &pay.Terms{}
	var dates []*pay.DueDate

	if settlement.PaymentTerms != nil {
		if settlement.PaymentTerms.Description != "" {
			terms.Detail = settlement.PaymentTerms.Description
		}

		if settlement.PaymentTerms.DueDate != nil && settlement.PaymentTerms.DueDate.DateFormat != nil {
			dueDateTime, err := parseDate(settlement.PaymentTerms.DueDate.DateFormat.Value)
			if err != nil {
				return nil, err
			}
			dd := &pay.DueDate{
				Date: &dueDateTime,
			}
			if len(dates) == 0 {
				p, err := num.PercentageFromString("100%")
				if err != nil {
					return nil, err
				}
				dd.Percent = &p
			}
			dates = append(dates, dd)
		}
	}
	terms.DueDates = dates

	if len(terms.DueDates) == 0 &&
		terms.Notes == "" &&
		terms.Key == "" &&
		terms.Detail == "" &&
		len(terms.Ext) == 0 {
		return nil, nil
	}

	return terms, nil
}

func goblNewInstructions(stlm *Settlement) *pay.Instructions {
	pm := stlm.PaymentMeans[0]
	inst := &pay.Instructions{
		Key: goblPaymentMeansCode(pm.TypeCode),
		Ext: tax.Extensions{
			untdid.ExtKeyPaymentMeans: cbc.Code(pm.TypeCode),
		},
	}

	if pm.Information != "" {
		inst.Detail = pm.Information
	}

	if pm.Card != nil {
		if pm.Card != nil {
			card := pm.Card
			inst.Card = &pay.Card{
				// GOBL only stores last 4 digits of card number
				Last4: card.ID[len(card.ID)-4:],
			}
			if card.Name != "" {
				inst.Card.Holder = card.Name
			}
		}
	}

	if pm.Creditor != nil {
		ac := pm.Creditor
		ct := new(pay.CreditTransfer)
		if ac.IBAN != "" {
			ct.IBAN = ac.IBAN
		}
		if ac.Name != "" {
			ct.Name = ac.Name
		}
		if ac.Number != "" {
			ct.Number = ac.Number
		}
		if pm.CreditorInstitution != nil && pm.CreditorInstitution.BIC != "" {
			ct.BIC = pm.CreditorInstitution.BIC
		}
		inst.CreditTransfer = append(inst.CreditTransfer, ct)
	}
	return inst
}

// paymentMeansCode maps a CII payment means to a GOBL equivalent
// Source https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred4461.htm
func goblPaymentMeansCode(code string) cbc.Key {
	if val, ok := paymentMeansMap[code]; ok {
		return val
	}
	return pay.MeansKeyAny
}
