package ctog

import (
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) preparePayment(stlm *document.Settlement) error {
	pymt := &bill.Payment{}

	if stlm.Payee != nil {
		payee := &org.Party{Name: stlm.Payee.Name}
		if stlm.Payee.PostalTradeAddress != nil {
			payee.Addresses = []*org.Address{
				parseAddress(stlm.Payee.PostalTradeAddress),
			}
		}
		pymt.Payee = payee
	}
	if len(stlm.PaymentTerms) > 0 {
		terms, err := getTerms(stlm)
		if err != nil {
			return err
		}
		pymt.Terms = terms
	}

	if len(stlm.PaymentMeans) > 0 && stlm.PaymentMeans[0].TypeCode != "1" {
		pymt.Instructions = getMeans(stlm)
	}

	if len(stlm.Advance) > 0 {
		for _, ap := range stlm.Advance {
			amt, err := num.AmountFromString(ap.Amount)
			if err != nil {
				return err
			}
			a := &pay.Advance{
				Amount: amt,
			}
			if ap.Date != nil && ap.Date.DateFormat != nil {
				advancePaymentReceivedDateTime, err := ParseDate(ap.Date.DateFormat.Value)
				if err != nil {
					return err
				}
				a.Date = &advancePaymentReceivedDateTime
			}
			pymt.Advances = append(pymt.Advances, a)
		}
	}

	c.inv.Payment = pymt
	return nil
}

func getTerms(settlement *document.Settlement) (*pay.Terms, error) {
	terms := &pay.Terms{}
	var dates []*pay.DueDate

	for _, term := range settlement.PaymentTerms {
		if term.Description != "" {
			terms.Detail = term.Description
		}

		if term.DueDate != nil && term.DueDate.DateFormat != nil {
			dueDateTime, err := ParseDate(term.DueDate.DateFormat.Value)
			if err != nil {
				return nil, err
			}
			dd := &pay.DueDate{
				Date: &dueDateTime,
			}
			if term.PartialPayment != "" {
				amt, err := num.AmountFromString(term.PartialPayment)
				if err != nil {
					return nil, err
				}
				dd.Amount = amt
			} else if len(dates) == 0 {
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
	return terms, nil
}

func getMeans(stlm *document.Settlement) *pay.Instructions {
	pm := stlm.PaymentMeans[0]
	inst := &pay.Instructions{
		Key: paymentMeansCode(pm.TypeCode),
		Ext: tax.Extensions{
			untdid.ExtKeyPaymentMeans: tax.ExtValue(pm.TypeCode),
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
		if ac.IBAN != "" {
			inst.CreditTransfer = []*pay.CreditTransfer{
				{
					IBAN: ac.IBAN,
				},
			}
		}
		if ac.Name != "" {
			inst.CreditTransfer[0].Name = ac.Name
		}
		if pm.CreditorInstitution != nil && pm.CreditorInstitution.BIC != "" {
			inst.CreditTransfer[0].BIC = pm.CreditorInstitution.BIC
		}
	}
	return inst
}
