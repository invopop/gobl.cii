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
func NewSettlement(inv *bill.Invoice) *Settlement {
	settlement := &Settlement{
		Currency: string(inv.Currency),
	}
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
		settlement.Means = &PaymentMeans{
			TypeCode:    findPaymentKey(inv.Payment.Instructions.Key),
			Information: inv.Payment.Instructions.Detail,
		}
		if inv.Payment.Instructions.CreditTransfer != nil {
			settlement.Means.Creditor = &Creditor{
				IBAN:   inv.Payment.Instructions.CreditTransfer[0].IBAN,
				Name:   inv.Payment.Instructions.CreditTransfer[0].Name,
				Number: inv.Payment.Instructions.CreditTransfer[0].Number,
			}
		}
		if inv.Payment.Instructions.DirectDebit != nil {
			settlement.Means.Debtor = inv.Payment.Instructions.DirectDebit.Account
			settlement.CreditorRefID = inv.Payment.Instructions.DirectDebit.Creditor
			if settlement.PaymentTerms == nil {
				settlement.PaymentTerms = new(Terms)
			}
			settlement.PaymentTerms.Mandate = inv.Payment.Instructions.DirectDebit.Ref
		}
		if inv.Payment.Instructions.Card != nil {
			settlement.Means.Card = &Card{
				ID:   inv.Payment.Instructions.Card.Last4,
				Name: inv.Payment.Instructions.Card.Holder,
			}
		}
	}

	return settlement
}

func newSummary(totals *bill.Totals, currency string) *Summary {
	return &Summary{
		TotalAmount:         totals.Total.String(),
		TaxBasisTotalAmount: totals.Total.String(),
		GrandTotalAmount:    totals.TotalWithTax.String(),
		DuePayableAmount:    totals.Payable.String(),
		TaxTotalAmount: &TaxTotalAmount{
			Amount:   totals.Tax.String(),
			Currency: currency,
		},
	}
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
		CategoryCode:          FindTaxCode(rate.Key),
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
