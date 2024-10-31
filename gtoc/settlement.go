package gtoc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

// One GOBL Release, update this to use catalogues
var paymentMeans = map[cbc.Key]string{
	pay.MeansKeyCash:           "10",
	pay.MeansKeyCheque:         "20",
	pay.MeansKeyCreditTransfer: "30",
	pay.MeansKeyCard:           "48",
	pay.MeansKeyDirectDebit:    "49",
	// pay.MeansKeyCreditTransfer.With(pay.MeansKeySEPA): "58",
	// pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA):    "59",
}

// NewSettlement creates the ApplicableHeaderTradeSettlement part of a EN 16931 compliant invoice
func (c *Converter) NewSettlement(inv *bill.Invoice) error {
	c.doc.Transaction.Settlement = &Settlement{
		Currency: string(inv.Currency),
	}
	settlement := c.doc.Transaction.Settlement
	if inv.Payment != nil && inv.Payment.Terms != nil {
		settlement.PaymentTerms = &Terms{
			Description: inv.Payment.Terms.Detail,
		}
	}

	if inv.Totals != nil {
		settlement.Tax = newTaxes(inv.Totals.Taxes)
		settlement.Summary = newSummary(inv.Totals, string(inv.Currency))
	}

	if len(inv.Preceding) > 0 {
		cor := inv.Preceding[0]
		settlement.ReferencedDocument = &ReferencedDocument{
			IssuerAssignedID: invoiceNumber(cor.Series, cor.Code),
			IssueDate: &Date{
				Date:   formatIssueDate(*cor.IssueDate),
				Format: "102",
			},
		}
	}
	if inv.Payment != nil && inv.Payment.Payee != nil {
		settlement.Payee = newPayee(inv.Payment.Payee)
	}

	if inv.Delivery != nil && inv.Delivery.Period != nil {
		settlement.Period = &Period{
			Start: &Date{
				Date:   formatIssueDate(inv.Delivery.Period.Start),
				Format: "102",
			},
			End: &Date{
				Date:   formatIssueDate(inv.Delivery.Period.End),
				Format: "102",
			},
		}
	}

	if inv.Payment != nil && inv.Payment.Instructions != nil {
		instr := inv.Payment.Instructions
		settlement.PaymentMeans = &PaymentMeans{
			TypeCode:    findPaymentKey(instr.Key),
			Information: instr.Detail,
		}
		if instr.CreditTransfer != nil {
			settlement.PaymentMeans.Creditor = &Creditor{
				IBAN:   instr.CreditTransfer[0].IBAN,
				Name:   instr.CreditTransfer[0].Name,
				Number: instr.CreditTransfer[0].Number,
			}
			if instr.CreditTransfer[0].BIC != "" {
				settlement.PaymentMeans.BICID = instr.CreditTransfer[0].BIC
			}
		}
		if instr.DirectDebit != nil {
			settlement.PaymentMeans.Debtor = instr.DirectDebit.Account
			settlement.CreditorRefID = instr.DirectDebit.Creditor
			if settlement.PaymentTerms == nil {
				settlement.PaymentTerms = new(Terms)
			}
			settlement.PaymentTerms.Mandate = instr.DirectDebit.Ref
		}
		if instr.Card != nil {
			settlement.PaymentMeans.Card = &Card{
				ID:   instr.Card.Last4,
				Name: instr.Card.Holder,
			}
		}
	}

	if len(inv.Charges) > 0 || len(inv.Discounts) > 0 {
		settlement.AllowanceCharges = newAllowanceCharges(inv)
	}

	return nil
}

func newSummary(totals *bill.Totals, currency string) *Summary {
	s := &Summary{
		TotalAmount:         totals.Total.String(),
		TaxBasisTotalAmount: totals.Total.String(),
		GrandTotalAmount:    totals.TotalWithTax.String(),
		DuePayableAmount:    totals.Payable.String(),
		TaxTotalAmount: &TaxTotalAmount{
			Amount:   totals.Tax.String(),
			Currency: currency,
		},
	}

	if totals.Charge != nil {
		s.Charges = totals.Charge.String()
	}

	if totals.Discount != nil {
		s.Discounts = totals.Discount.String()
	}

	return s
}

func newTaxes(total *tax.Total) []*Tax {
	var Taxes []*Tax

	if total == nil {
		return nil
	}

	for _, category := range total.Categories {
		for _, rate := range category.Rates {
			tax := newTax(rate, category)

			Taxes = append(Taxes, tax)
		}
	}

	return Taxes
}

func newTax(rate *tax.RateTotal, category *tax.CategoryTotal) *Tax {
	if rate.Percent == nil {
		return nil
	}

	tax := &Tax{
		CalculatedAmount:      rate.Amount.String(),
		TypeCode:              category.Code.String(),
		BasisAmount:           rate.Base.String(),
		CategoryCode:          findTaxCode(rate.Key),
		RateApplicablePercent: rate.Percent.StringWithoutSymbol(),
	}

	return tax
}

func newPayee(party *org.Party) *Buyer {
	payee := &Buyer{
		Name:                      party.Name,
		Contact:                   newContact(party),
		PostalTradeAddress:        NewPostalTradeAddress(party.Addresses),
		URIUniversalCommunication: NewEmail(party.Emails),
	}

	if party.TaxID != nil {
		payee.ID = party.TaxID.String()
	}

	return payee
}

func findPaymentKey(key cbc.Key) string {
	if val, ok := paymentMeans[key]; ok {
		return val
	}
	return "1"
}
