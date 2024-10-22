package gtoc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
)

// NewSettlement creates the ApplicableHeaderTradeSettlement part of a EN 16931 compliant invoice
func NewSettlement(inv *bill.Invoice) *Settlement {
	settlement := &Settlement{
		Currency: string(inv.Currency),
		TypeCode: FindTypeCode(inv),
	}
	if inv.Payment != nil && inv.Payment.Terms != nil {
		settlement.PaymentTerms = inv.Payment.Terms.Detail
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
