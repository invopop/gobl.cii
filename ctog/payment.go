package ctog

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
)

func (c *Converter) getPayment(settlement *ApplicableHeaderTradeSettlement) error {
	payment := &bill.Payment{}

	if settlement.PayeeTradeParty != nil {
		payee := &org.Party{Name: settlement.PayeeTradeParty.Name}
		if settlement.PayeeTradeParty.PostalTradeAddress != nil {
			payee.Addresses = []*org.Address{
				parseAddress(settlement.PayeeTradeParty.PostalTradeAddress),
			}
		}
		payment.Payee = payee
	}
	if len(settlement.SpecifiedTradePaymentTerms) > 0 {
		if settlement.SpecifiedTradePaymentTerms[0].DueDateDateTime != nil {
			terms, err := getTerms(settlement)
			if err != nil {
				return err
			}
			payment.Terms = terms
		}
	}

	if len(settlement.SpecifiedTradeSettlementPaymentMeans) > 0 && settlement.SpecifiedTradeSettlementPaymentMeans[0].TypeCode != "1" {
		payment.Instructions = getMeans(settlement)
	}

	if len(settlement.SpecifiedAdvancePayment) > 0 {
		for _, advancePayment := range settlement.SpecifiedAdvancePayment {
			advance := &pay.Advance{
				Amount: num.AmountFromFloat64(advancePayment.PaidAmount, 0),
			}
			if advancePayment.FormattedReceivedDateTime != nil {
				advancePaymentReceivedDateTime, err := ParseDate(advancePayment.FormattedReceivedDateTime.DateTimeString)
				if err != nil {
					return err
				}
				advance.Date = &advancePaymentReceivedDateTime
			}
			payment.Advances = append(payment.Advances, advance)
		}
	}

	c.inv.Payment = payment
	return nil
}

func getTerms(settlement *ApplicableHeaderTradeSettlement) (*pay.Terms, error) {
	terms := &pay.Terms{}
	var dueDates []*pay.DueDate

	for _, paymentTerm := range settlement.SpecifiedTradePaymentTerms {
		if paymentTerm.Description != nil {
			terms.Detail = *paymentTerm.Description
		}

		if paymentTerm.DueDateDateTime != nil {
			dueDateTime, err := ParseDate(paymentTerm.DueDateDateTime.DateTimeString)
			if err != nil {
				return nil, err
			}
			dueDate := &pay.DueDate{
				Date: &dueDateTime,
			}
			if paymentTerm.PartialPaymentAmount != nil {
				dueDate.Amount, _ = num.AmountFromString(*paymentTerm.PartialPaymentAmount)
			} else if len(dueDates) == 0 {
				percent, err := num.PercentageFromString("100%")
				if err != nil {
					return nil, err
				}
				dueDate.Percent = &percent
			}
			dueDates = append(dueDates, dueDate)
		}
	}
	terms.DueDates = dueDates
	return terms, nil
}

func getMeans(settlement *ApplicableHeaderTradeSettlement) *pay.Instructions {
	paymentMeans := settlement.SpecifiedTradeSettlementPaymentMeans[0]
	instructions := &pay.Instructions{
		Key: PaymentMeansTypeCodeParse(paymentMeans.TypeCode),
	}

	if paymentMeans.Information != nil {
		instructions.Detail = *paymentMeans.Information
	}

	if paymentMeans.ApplicableTradeSettlementFinancialCard != nil {
		if paymentMeans.ApplicableTradeSettlementFinancialCard != nil {
			card := paymentMeans.ApplicableTradeSettlementFinancialCard
			instructions.Card = &pay.Card{
				// GOBL only stores last 4 digits of card number
				Last4: card.ID[len(card.ID)-4:],
			}
			if card.CardholderName != "" {
				instructions.Card.Holder = card.CardholderName
			}
		}
	}

	if paymentMeans.PayeePartyCreditorFinancialAccount != nil {
		account := paymentMeans.PayeePartyCreditorFinancialAccount
		if account.IBANID != "" {
			instructions.CreditTransfer = []*pay.CreditTransfer{
				{
					IBAN: account.IBANID,
				},
			}
		}
		if account.AccountName != "" {
			instructions.CreditTransfer[0].Name = account.AccountName
		}
		if paymentMeans.PayeeSpecifiedCreditorFinancialInstitution != nil {
			instructions.CreditTransfer[0].BIC = paymentMeans.PayeeSpecifiedCreditorFinancialInstitution.BICID
		}
	}
	return instructions
}
