package ctog

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
)

func (c *Converter) preparePayment(stlm *ApplicableHeaderTradeSettlement) error {
	pymt := &bill.Payment{}

	if stlm.PayeeTradeParty != nil {
		payee := &org.Party{Name: stlm.PayeeTradeParty.Name}
		if stlm.PayeeTradeParty.PostalTradeAddress != nil {
			payee.Addresses = []*org.Address{
				parseAddress(stlm.PayeeTradeParty.PostalTradeAddress),
			}
		}
		pymt.Payee = payee
	}
	if len(stlm.SpecifiedTradePaymentTerms) > 0 {
		if stlm.SpecifiedTradePaymentTerms[0].DueDateDateTime != nil {
			terms, err := getTerms(stlm)
			if err != nil {
				return err
			}
			pymt.Terms = terms
		}
	}

	if len(stlm.SpecifiedTradeSettlementPaymentMeans) > 0 && stlm.SpecifiedTradeSettlementPaymentMeans[0].TypeCode != "1" {
		pymt.Instructions = getMeans(stlm)
	}

	if len(stlm.SpecifiedAdvancePayment) > 0 {
		for _, ap := range stlm.SpecifiedAdvancePayment {
			a := &pay.Advance{
				Amount: num.AmountFromFloat64(ap.PaidAmount, 0),
			}
			if ap.FormattedReceivedDateTime != nil {
				advancePaymentReceivedDateTime, err := ParseDate(ap.FormattedReceivedDateTime.DateTimeString)
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

func getTerms(settlement *ApplicableHeaderTradeSettlement) (*pay.Terms, error) {
	terms := &pay.Terms{}
	var dates []*pay.DueDate

	for _, term := range settlement.SpecifiedTradePaymentTerms {
		if term.Description != nil {
			terms.Detail = *term.Description
		}

		if term.DueDateDateTime != nil {
			dueDateTime, err := ParseDate(term.DueDateDateTime.DateTimeString)
			if err != nil {
				return nil, err
			}
			dd := &pay.DueDate{
				Date: &dueDateTime,
			}
			if term.PartialPaymentAmount != nil {
				dd.Amount, err = num.AmountFromString(*term.PartialPaymentAmount)
				if err != nil {
					return nil, err
				}
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

func getMeans(stlm *ApplicableHeaderTradeSettlement) *pay.Instructions {
	pm := stlm.SpecifiedTradeSettlementPaymentMeans[0]
	inst := &pay.Instructions{
		Key: PaymentMeansTypeCodeParse(pm.TypeCode),
	}

	if pm.Information != nil {
		inst.Detail = *pm.Information
	}

	if pm.ApplicableTradeSettlementFinancialCard != nil {
		if pm.ApplicableTradeSettlementFinancialCard != nil {
			card := pm.ApplicableTradeSettlementFinancialCard
			inst.Card = &pay.Card{
				// GOBL only stores last 4 digits of card number
				Last4: card.ID[len(card.ID)-4:],
			}
			if card.CardholderName != "" {
				inst.Card.Holder = card.CardholderName
			}
		}
	}

	if pm.PayeePartyCreditorFinancialAccount != nil {
		ac := pm.PayeePartyCreditorFinancialAccount
		if ac.IBANID != "" {
			inst.CreditTransfer = []*pay.CreditTransfer{
				{
					IBAN: ac.IBANID,
				},
			}
		}
		if ac.AccountName != "" {
			inst.CreditTransfer[0].Name = ac.AccountName
		}
		if pm.PayeeSpecifiedCreditorFinancialInstitution != nil {
			inst.CreditTransfer[0].BIC = pm.PayeeSpecifiedCreditorFinancialInstitution.BICID
		}
	}
	return inst
}
